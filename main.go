package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"slices"
	"strings"
)

const (
	maxContentLength = 2000
	maxFiles         = 10
)

func main() {
	// フラグの定義
	var filePaths stringSlice
	flag.Var(&filePaths, "f", "ファイルパスを指定")
	flag.Parse()

	discordWebHookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if discordWebHookURL == "" {
		fmt.Println("環境変数 DISCORD_WEBHOOK_URL が設定されていません")
		return
	}

	var body io.Reader
	var req *http.Request

	// -f フラグが指定されている場合はファイルをマルチパート形式で送信
	if len(filePaths) > 0 {
		for chunk := range slices.Chunk(filePaths, maxFiles) {
			// マルチパートのバッファを作成
			var requestBody bytes.Buffer
			writer := multipart.NewWriter(&requestBody)

			for i, filePath := range chunk {
				file, err := os.Open(filePath)
				if err != nil {
					panic(err)
				}
				defer file.Close()

				// ファイルをマルチパートとして追加
				part, err := writer.CreateFormFile(fmt.Sprintf("file[%d]", i), filePath)
				if err != nil {
					panic(err)
				}
				_, err = io.Copy(part, file)
				if err != nil {
					panic(err)
				}
			}

			// フォームデータの完了を知らせる
			err := writer.Close()
			if err != nil {
				panic(err)
			}

			// リクエスト作成
			req, err := http.NewRequest("POST", discordWebHookURL, &requestBody)
			if err != nil {
				panic(err)
			}

			// マルチパートの境界をContent-Typeヘッダーに設定
			req.Header.Set("Content-Type", writer.FormDataContentType())

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()

			ret, err := io.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}
			fmt.Print(string(ret))
		}

	} else {
		var content string

		// 引数が指定されている場合はその内容を使用
		if flag.NArg() > 0 {
			content = strings.Join(flag.Args(), " ")
		} else {
			// 標準入力から内容を読み込む
			stdinContent, err := io.ReadAll(os.Stdin)
			if err != nil {
				panic(err)
			}
			content = string(stdinContent)
		}

		// 改行や特殊文字を適切にエスケープ
		content = strings.TrimSpace(content)

		// 2000文字を超える場合は分割して送信
		contents := splitContentAtNewline(content, maxContentLength)

		for _, c := range contents {
			jsonContent, err := json.Marshal(map[string]string{"content": c})
			if err != nil {
				panic(err)
			}

			body = bytes.NewReader(jsonContent)
			req, err = http.NewRequest("POST", discordWebHookURL, body)
			if err != nil {
				panic(err)
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()

			ret, err := io.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}
			fmt.Print(string(ret))
		}
	}
}

// コンテンツを指定した長さ以内で最後の改行を基に分割する関数
func splitContentAtNewline(content string, length int) []string {
	var result []string
	runes := []rune(content) // マルチバイト文字を考慮するためruneに変換

	for len(runes) > length {
		// 最初の2000文字以内で最後の改行位置を探す
		slice := runes[:length]
		lastNewline := strings.LastIndex(string(slice), "\n")

		if lastNewline == -1 {
			// 改行がなければ2000文字で切る
			lastNewline = length
		}

		// 分割してリストに追加
		result = append(result, string(runes[:lastNewline]))
		// 残りの文字列に進む
		runes = runes[lastNewline:]
	}

	result = append(result, string(runes))
	return result
}

type stringSlice []string

func (s stringSlice) String() string {
	return strings.Join(s, ", ")
}

func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}
