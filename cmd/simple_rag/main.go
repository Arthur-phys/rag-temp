package main

import (
	"context"
	"fmt"
	"log"
	"ragchat/internal/chunks"

	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

func main() {
	llm, err := ollama.New(ollama.WithModel("llama3.2"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Creating embedder")
	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Creating chunks")
	newChunks, err := chunks.NewFromMarkdown("./testfiles/dinosaurs.md", 500, 50)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Chunks:", len(newChunks))
	embs, err := embedder.EmbedDocuments(context.Background(), newChunks)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Embs:", len(embs))

	conn, err := pgx.Connect(context.Background(), "postgres://ragger:ragger@localhost:5432/rag_retrieval")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		log.Fatal(err)
	}
	err = pgxvec.RegisterTypes(context.Background(), conn)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserting chunks:")
	for i, emb := range embs {
		_, err = conn.Exec(context.Background(), "INSERT INTO text_embedding (content, embedding) VALUES ($1, $2)", newChunks[i], pgvector.NewVector(emb))
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("Making a question:", "What are the currently agreed upon general dinosaur characteristics?")
	userChunk, err := embedder.EmbedQuery(context.Background(), "What are the currently agreed upon general dinosaur characteristics?")
	if err != nil {
		log.Fatal(err)
	}
	var textContext string
	rows, err := conn.Query(context.Background(), "SELECT content FROM text_embedding ORDER BY embedding <-> $1 LIMIT 5", pgvector.NewVector(userChunk))
	if err != nil {
		log.Fatal(err)
	}
	decodedRow, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		log.Fatal(err)
	}
	for _, strRow := range decodedRow {
		textContext = textContext + strRow + "\n"
	}
	rows.Close()

	format := fmt.Sprintf("Use the following context to answer the question. If you don't know the answer, simply say I dont know: Context:\n%v\nQuestion:\n%v", textContext, "What are the currently agreed upon general dinosaur characteristics?")
	completion, err := llms.GenerateFromSinglePrompt(context.Background(), llm, format)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:\n", completion)

}
