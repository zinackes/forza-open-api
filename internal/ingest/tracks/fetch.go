package tracks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// userAgent identifie l'ingestion auprès des sources (politesse exigée par la
// doctrine : user-agent identifiable, lecture seule).
const userAgent = "forza-open-api-ingestion/0.1 (+https://github.com/zinackes/forza-open-api)"

var httpc = &http.Client{Timeout: 20 * time.Second}

// FetchDataset lit un dataset de tracés depuis un chemin local ou une URL http(s)
// (dataset communautaire GitHub, export wiki Fandom…). Lecture seule, timeout et
// user-agent identifiable. La taille est bornée pour éviter un OOM.
func FetchDataset(ctx context.Context, src string) ([]byte, error) {
	if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
		raw, err := os.ReadFile(src)
		if err != nil {
			return nil, fmt.Errorf("read dataset %s: %w", src, err)
		}
		return raw, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", src, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %d", src, resp.StatusCode)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("read body %s: %w", src, err)
	}
	return raw, nil
}
