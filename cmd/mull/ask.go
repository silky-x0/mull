package main

import (
	"errors"
	"os"
	"strings"
)

const askSystem = "You are a knowledgeable assistant. Answer the question accurately and concisely. Output only the answer, with no preamble or questions back."

func ask(verbose bool, args []string) error {
	q, err := askQuestion(args)
	if err != nil {
		return err
	}
	attempts, err := requestBackends()
	if err != nil {
		return err
	}
	ctx, cancel := requestContext()
	defer cancel()
	return complete(ctx, attempts, askSystem, q, verbose, checkNonEmpty, os.Stdout, os.Stderr)
}

func askQuestion(args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("ask needs a question, e.g. mull ask \"what is a goroutine?\"")
	}
	return strings.Join(args, " "), nil
}
