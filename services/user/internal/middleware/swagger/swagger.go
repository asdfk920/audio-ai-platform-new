package swagger

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	swaggerUIURL = "https://unpkg.com/swagger-ui-dist@5.11.0"
)

func SwaggerHandler(docPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/swagger/")

		switch path {
		case "", "index.html", "doc.html":
			renderSwaggerUI(w, r)
		case "doc.json":
			serveDocJSON(w, r, docPath)
		default:
			if strings.HasPrefix(path, "http") {
				proxyAsset(w, r, path)
			} else {
				http.NotFound(w, r)
			}
		}
	}
}

func serveDocJSON(w http.ResponseWriter, r *http.Request, docPath string) {
	file, err := os.Open(docPath)
	if err != nil {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = io.Copy(w, file)
}

func renderSwaggerUI(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>用户微服务 API 文档</title>
    <link rel="stylesheet" type="text/css" href="` + swaggerUIURL + `/swagger-ui.css">
    <style>
        body { margin: 0; padding: 20px; background: #fafafa; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
        .header { text-align: center; margin-bottom: 30px; padding: 20px; background: white; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .header h1 { color: #333; margin: 0 0 10px 0; font-size: 28px; }
        .header p { color: #666; margin: 0; font-size: 14px; }
        .info-box { background: #e8f4fd; border-left: 4px solid #2196F3; padding: 15px; margin: 20px 0; border-radius: 4px; }
        .info-box strong { color: #1976D2; }
        .stats { display: flex; justify-content: space-around; margin: 20px 0; flex-wrap: wrap; gap: 10px; }
        .stat-card { background: white; padding: 15px 25px; border-radius: 8px; box-shadow: 0 2px 6px rgba(0,0,0,0.08); text-align: center; min-width: 120px; }
        .stat-number { font-size: 24px; font-weight: bold; color: #2196F3; }
        .stat-label { font-size: 12px; color: #666; margin-top: 5px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🚀 用户微服务 API</h1>
        <p>User Microservice API Documentation | Version 1.0.0</p>
    </div>

    <div class="stats">
        <div class="stat-card">
            <div class="stat-number">23</div>
            <div class="stat-label">API 接口</div>
        </div>
        <div class="stat-card">
            <div class="stat-number">5</div>
            <div class="stat-label">业务模块</div>
        </div>
        <div class="stat-card">
            <div class="stat-number">100%</div>
            <div class="stat-label">文档覆盖</div>
        </div>
    </div>

    <div class="info-box">
        <strong>📌 核心接口：</strong> 设备共享（发起/接受/拒绝/撤销/退出）| 设备管理（绑定/列表/详情）<br>
        <strong>🔐 认证方式：</strong> Bearer Token (JWT) | <strong>📡 服务地址：</strong> localhost:8001
    </div>

    <div id="swagger-ui"></div>
    <script src="` + swaggerUIURL + `/swagger-ui-bundle.js"></script>
    <script src="` + swaggerUIURL + `/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "/swagger/doc.json",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                layout: "StandaloneLayout",
                filter: true,
                requestInterceptor: (req) => {
                    const token = localStorage.getItem('token');
                    if (token) {
                        req.headers.Authorization = 'Bearer ' + token;
                    }
                    return req;
                },
                displayRequestDuration: true,
                tryItOutEnabled: true
            });

            window.ui = ui;
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

func proxyAsset(w http.ResponseWriter, r *http.Request, assetURL string) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", assetURL, nil)

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to load asset", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	contentType := ""
	ext := filepath.Ext(assetURL)
	switch ext {
	case ".css":
		contentType = "text/css"
	case ".js":
		contentType = "application/javascript"
	case ".png":
		contentType = "image/png"
	default:
		contentType = resp.Header.Get("Content-Type")
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(resp.StatusCode)

	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}
