package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/openai/openai-go"
)

func checkIfValidImage(fileName string, fileSize int64) error {
	if fileSize/1000000 > 20 {
		return errors.New("file too large. Files must be 20MB and less")
	}
	if filepath.Ext(fileName) != ".png" && filepath.Ext(fileName) != ".jpg" && filepath.Ext(fileName) != ".jpeg" {

		return errors.New("invalid image type. Please upload either a .png, .jpg, or .jpeg file")
	}

	return nil

}

/** OPEN AI UTILS*/

type ErrorDetails struct {
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Param   interface{} `json:"param"`
	Code    interface{} `json:"code"`
}
type OpenAIError struct {
	Error ErrorDetails `json:"error"`
}

func handleOpenAIError(apiErr *openai.Error, bankRes bankResponse) (bankResponse, error) {
	var openAiErr OpenAIError
	err := json.Unmarshal([]byte(apiErr.JSON.RawJSON()), &openAiErr)
	if err != nil {
		return bankRes, errors.New("env: OPENAI_API_KEY not found")
	}
	fmt.Print("error unmarshalled", openAiErr)

	return bankRes, errors.New(openAiErr.Error.Message)
}
