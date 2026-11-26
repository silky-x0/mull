package main

import "os"

const summarizeSystem = "You are a text summarizer. Summarize the text concisely in its own language. Output only the summary, without notes or explanations."

func summarize(verbose, stream bool) error {
	input, err := readInput(os.Stdin)
	if err != nil {
		return err
	}
	attempts, err := requestBackends()
	if err != nil {
		return err
	}
	ctx, cancel := requestContext()
	defer cancel()
	return complete(ctx, attempts, summarizeSystem, summarizeMessage(input), verbose, stream, checkNonEmpty, os.Stdout, os.Stderr)
}

func summarizeMessage(input string) string {
	return rewriteMessage("Summarize the text.", input)
}
