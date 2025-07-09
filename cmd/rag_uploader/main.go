package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/pgvector"
)

func main() {

	var fileLocation string
	var profile string
	fmt.Print("> Write the name of your profile: ")
	fmt.Scanln(&profile)
	fmt.Print("> Write the route to your document: ")
	fmt.Scanln(&fileLocation)

	fileRe := regexp.MustCompile(`.*\.([a-z]+)$`)
	matches := fileRe.FindStringSubmatch(fileLocation)
	if len(matches) <= 1 {
		log.Fatal("The document given has no extension!")
		return
	}
	file, err := os.Open(fileLocation)
	if err != nil {
		log.Fatal(err)
		return
	}

	llm, err := ollama.New(ollama.WithModel("llama3.2"))
	if err != nil {
		log.Fatal(err)
		return
	}
	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatal(err)
		return
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
		pgvector.WithPreDeleteCollection(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	var documents []schema.Document
	options := []textsplitter.Option{textsplitter.WithChunkOverlap(100), textsplitter.WithChunkSize(500)}
	switch matches[1] {
	case "md":
		langChainText := documentloaders.NewText(file)
		documents, err = langChainText.LoadAndSplit(context.Background(), textsplitter.NewMarkdownTextSplitter(options...))
	case "html":
		langChainText := documentloaders.NewHTML(file)
		// No explicit splitter for html... Why?
		// This splitter should change for each type of file
		documents, err = langChainText.LoadAndSplit(context.Background(), textsplitter.NewMarkdownTextSplitter(options...))
	case "pdf":
		langChainText := documentloaders.NewPDF(file, 0)
		// Also no splitter for pdf
		documents, err = langChainText.LoadAndSplit(context.Background(), textsplitter.NewMarkdownTextSplitter(options...))
	default:
		langChainText := documentloaders.NewText(file)
		documents, err = langChainText.LoadAndSplit(context.Background(), textsplitter.NewMarkdownTextSplitter(options...))
	}

	if err != nil {
		log.Fatal(err)
	}
	ids, err := store.AddDocuments(context.Background(), documents)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Print("Here are the ids of the created documents: ")
	for _, id := range ids {
		log.Printf("%v", id)
	}
	log.Println("")

	log.Println("Finished uploading to vector store")
}
