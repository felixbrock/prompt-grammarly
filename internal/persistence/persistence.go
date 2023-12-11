package persistence

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/felixbrock/lemonai/internal/domain"
)

// https://cllevlrokigwvbbnbfiu.supabase.co

func Write(optimization domain.Optimization) error {
	file, err := os.OpenFile("optimization.csv", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	defer func() {
		err = file.Close()
		if err != nil {
			slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		}
	}()

	writer := csv.NewWriter(file)

	defer func() {
		writer.Flush()
		err = writer.Error()
		if err != nil {
			slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		}
	}()

	record := []string{
		optimization.Id,
		optimization.Prompt,
		optimization.Instructions,
		optimization.ParentId,
	}

	err = writer.Write(record)
	if err != nil {
		return err
	}

	return nil
}

func toOptimization(record []string) domain.Optimization {
	return domain.Optimization{
		Id:           record[0],
		Prompt:       record[1],
		Instructions: record[2],
		ParentId:     record[3],
	}
}

func Read(id string) (*domain.Optimization, error) {
	file, err := os.Open("optimization.csv")
	if err != nil {
		return nil, err
	}

	defer func() {
		err = file.Close()
		if err != nil {
			slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		}
	}()

	reader := csv.NewReader(file)

	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if record[0] == id {
			optimization := toOptimization(record)
			return &optimization, err
		}
	}

	return nil, err
}
