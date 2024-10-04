package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"context"

	"encoding/base64"

	"github.com/invopop/jsonschema"
)

type bankResponse struct {
	BankName      string `json:"bank_name" jsonschema_description:"The name of the bank"`
	AccountNumber string `json:"account_number" jsonschema_description:"The account number"`
}

func bankSuggestionsFromLiveText(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		ImageString string `json:"image_string"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		errorResponse(w, 400, fmt.Sprintf("Error parsing JSON: %v", err))
		return
	}

	if len(params.ImageString) > 300 {
		errorResponse(w, 400, "Content overload. Please scan a picture of just the bank and account number.")
		return
	}
	prompt := fmt.Sprintf("You are a helpful, friendly assistant. Extract an 11 digit number from the string: %v. \nEnsure there are no spaces between the 10 digits in the string then make it the value of the AccountNumber response. Then search the same string %v for a name that matches any of the banks in the comma seppatated list of banks:%v. Account for possible mispellings in the string %v and find a bank that matches one of the strings. Note that any bank name could be abbreviated from the bank list. Check for possible abbreviations in the string as well. For example, the string might have Guaranty Trust Bank but might be GTB in the list of banks. The result should be extracted from the comma separated bank list and put as the BankName and AccountNumber. Make sure to check that your computation is right. If there is no matching bank account or number in the image that can match after multiple times of checking through the image, return an empty string as the values for AccountNumber and BankName. Do not hallucinate.", params.ImageString, params.ImageString, bankResultString, params.ImageString)
	promptObject := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		}),
		ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
			openai.ResponseFormatJSONSchemaParam{
				Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
				JSONSchema: openai.F(schemaParam),
			},
		),
		Model: openai.F(openai.ChatModelGPT4o2024_08_06),
	}
	data, err := askOpenAIForBank(promptObject)
	if err != nil {
		errorResponse(w, 500, fmt.Sprintf("There was an error processing this request. Please try again. %v", err))
		return
	}
	jsonResponse(w, 200, data)
}

func bankSuggestionsFromImage(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
	file, handler, err := r.FormFile("myFile")
	if err != nil {
		errorResponse(w, 400, fmt.Sprintf("Error processing file: %v", err))
		return
	}
	defer file.Close()
	fmt.Print(handler.Size, "file size")
	fmt.Printf("Uploaded file %+v\n", handler.Filename)
	fileErr := checkIfValidImage(handler.Filename, handler.Size)
	if fileErr != nil {
		errorResponse(w, 400, fileErr.Error())
		return
	}
	fileBytes, err := io.ReadAll(file)
	imageToBase64 := base64.StdEncoding.EncodeToString([]byte(fileBytes))

	if err != nil {
		errorResponse(w, 400, fmt.Sprintf("Could not read file: %v", err))
	}

	prompt := fmt.Sprintf("Your role is vision. Extract an 11 digit number from the image provided. Ensure there are no spaces between the 10 digits in the string then make it the value of the AccountNumber response. Then search the same image for a name that matches any of the banks in the comma sepatated list of banks:%v. Account for possible mispellings in the image and find a bank that matches one of the strings. Note that any bank name could be abbreviated and mispelled in the bank list but always return the bank list equivalent. Check for possible abbreviations in the image as well. For example, the image might have Guaranty Trust Bank but might be GTB in the list of banks. The result should be extracted from the comma separated bank list and put as the BankName. Make sure to check that your computation is right. If there is no matching bank account or number in the image that can match after multiple times of checking through the image, return an empty string as the values for AccountNumber and BankName. Do not hallucinate.", bankResultString)
	imageFormattedBase64 := fmt.Sprintf("data:image/jpeg;base64,%v", imageToBase64)
	promptObject := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessageParts(openai.ImagePart(imageFormattedBase64), openai.TextPart(prompt)),
		}),
		ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
			openai.ResponseFormatJSONSchemaParam{
				Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
				JSONSchema: openai.F(schemaParam),
			},
		),
		Model: openai.F(openai.ChatModelGPT4o2024_08_06),
	}
	data, err := askOpenAIForBank(promptObject)
	if err != nil {
		errorResponse(w, 500, "There was an error processing this request. Please try again.")
	}
	jsonResponse(w, 200, data)

}

func askOpenAIForBank(promptObject openai.ChatCompletionNewParams) (bankResponse, error) {
	var openAiKey, apiKeyPresent = os.LookupEnv("OPENAI_API_KEY")
	if !apiKeyPresent {
		return bankResponse{}, errors.New("env: OPENAI_API_KEY not found")
	}
	client := openai.NewClient(
		option.WithAPIKey(openAiKey),
	)
	chat, err := client.Chat.Completions.New(context.TODO(), promptObject)
	if err != nil {
		var apiErr *openai.Error
		if errors.As(err, &apiErr) {
			return handleOpenAIError(apiErr, bankResponse{})
		}

		return bankResponse{}, err
	}
	bankRes := bankResponse{}
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &bankRes)
	if err != nil {
		return bankResponse{}, err
	}
	return bankRes, nil

}

var bankResultString = strings.Join(AllBanks, ", ")

var schemaParam = openai.ResponseFormatJSONSchemaJSONSchemaParam{
	Name:        openai.F("nigerianbanks"),
	Description: openai.F("Get nigerian bank from string or image"),
	Schema:      openai.F(BankResponseSchema),
	Strict:      openai.Bool(true),
}

var BankResponseSchema = GenerateSchema[bankResponse]()

func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}
