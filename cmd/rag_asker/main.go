package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/vectorstores/pgvector"
)

const (
	TEMPLATE = `
	Role:
		You are a respectful assistant, always willing to help out those in need.
	Use the following context to answer the question. If you don't know the answer, simply say "I don't know"`
)

func main() {

	var profile string
	fmt.Print("> Write the name of your profile: ")
	fmt.Scanln(&profile)

	llm, err := ollama.New(ollama.WithModel("llama3.2"))
	if err != nil {
		log.Fatal(err)
	}
	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := pgx.Connect(context.Background(), "postgres://ragger:ragger@localhost:5432/rag_retrieval")
	if err != nil {
		log.Fatal(err)
		return
	}
	defer conn.Close(context.Background())

	store, err := pgvector.New(context.Background(),
		pgvector.WithConn(conn),
		pgvector.WithCollectionTableName("test_embeddings"),
		pgvector.WithCollectionName(profile),
		pgvector.WithEmbedder(embedder),
		pgvector.WithVectorDimensions(3072),
	)
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("> Introduce your question: ")
	userInput, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
		return
	}
	userInput = userInput[:len(userInput)-1]

	matchedDocuments, err := store.SimilaritySearch(context.Background(), userInput, 10)
	if err != nil {
		log.Fatal(err)
		return
	}
	textContext := TEMPLATE + "\nContext:\n"
	for _, document := range matchedDocuments {
		textContext = textContext + document.PageContent
	}
	textContext = textContext + "\nQuestion:\n" + userInput

	response, err := llms.GenerateFromSinglePrompt(context.Background(), llm, textContext)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Here is your answer: %v\n", response)

}
