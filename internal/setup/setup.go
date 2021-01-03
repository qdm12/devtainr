package setup

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func MirrorFile(ctx context.Context, client *http.Client,
	baseURL, basePath, filename string) (err error) {
	url := baseURL + "/" + filename
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	filepath := filepath.Join(basePath, filename)
	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	_, err = io.Copy(file, response.Body)
	if err != nil {
		_ = file.Close()
		return err
	}

	if err := response.Body.Close(); err != nil {
		_ = file.Close()
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}
