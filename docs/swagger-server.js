const http = require('http');
const fs = require('fs');
const path = require('path');

// Swagger UI HTML模板
const swaggerHTML = (title, specUrl) => `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>${title}</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css">
    <style>
        body { margin: 0; padding: 20px; background: #fafafa; }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 20px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        .header h1 { margin: 0; font-size: 2em; }
        .header p { margin: 10px 0 0 0; opacity: 0.9; }
        .info-box {
            background: white;
            padding: 15px;
            border-radius: 8px;
            margin-bottom: 15px;
            border-left: 4px solid #667eea;
        }
        .info-box h3 { margin-top: 0; color: #333; }
        .info-box code {
            background: #f4f4f4;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>📚 ${title}</h1>
        <p>音频AI平台 - 专业API接口文档</p>
    </div>

    <div id="swagger-ui"></div>

    <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "${specUrl}",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
                docExpansion: "list",
                filter: true,
                showRequestDuration: true
            });

            window.ui = ui;
        };
    </script>
</body>
</html>`;

class SwaggerServer {
    constructor(port, title, yamlPath) {
        this.port = port;
        this.title = title;
        this.yamlPath = yamlPath;
        this.server = null;
    }

    start() {
        this.server = http.createServer((req, res) => {
            if (req.url === '/' || req.url === '/index.html') {
                // 返回Swagger UI页面
                const html = swaggerHTML(this.title, `http://localhost:${this.port}/spec`);
                res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
                res.end(html);
            } else if (req.url === '/spec') {
                // 返回YAML规范文件
                try {
                    const content = fs.readFileSync(this.yamlPath, 'utf-8');
                    res.writeHead(200, {
                        'Content-Type': 'application/yaml;charset=utf-8',
                        'Access-Control-Allow-Origin': '*'
                    });
                    res.end(content);
                } catch (err) {
                    console.error(`❌ 读取文件失败: ${err.message}`);
                    res.writeHead(500, { 'Content-Type': 'text/plain' });
                    res.end('Internal Server Error');
                }
            } else {
                res.writeHead(404, { 'Content-Type': 'text/plain' });
                res.end('Not Found');
            }
        });

        this.server.listen(this.port, () => {
            console.log(`
╔══════════════════════════════════════════════════╗
║                                                  ║
║   🚀 ${this.title} 已启动!                      ║
║                                                  ║
║   📍 访问地址: http://localhost:${this.port}     ║
║   📄 文档路径: ${this.yamlPath}                  ║
║                                                  ║
║   💡 提示:                                       ║
║      - 按 Ctrl+C 停止服务器                       ║
║      - 浏览器自动刷新                             ║
║                                                  ║
╚══════════════════════════════════════════════════╝
            `);
        });

        this.server.on('error', (err) => {
            if (err.code === 'EADDRINUSE') {
                console.error(`❌ 端口 ${this.port} 已被占用，请检查是否有其他程序在使用该端口`);
            } else {
                console.error(`❌ 服务器启动失败: ${err.message}`);
            }
        });
    }

    stop() {
        if (this.server) {
            this.server.close();
            console.log(`\n✅ ${this.title} 已停止`);
        }
    }
}

// 获取命令行参数
const args = process.argv.slice(2);

if (args.length < 2) {
    console.log(`
╔══════════════════════════════════════════════════╗
║          📖 Swagger UI 本地预览工具               ║
╠══════════════════════════════════════════════════╣
║                                                  ║
║  使用方法:                                       ║
║    node swagger-server.js <port> <yaml-path>     ║
║                                                  ║
║  示例:                                           ║
║    node swagger-server.js 3001 ./swagger.yaml    ║
║                                                  ║
╚══════════════════════════════════════════════════╝
    `);
    process.exit(1);
}

const port = parseInt(args[0], 10);
const yamlPath = path.resolve(args[1]);
const title = args[2] || 'API Documentation';

if (!fs.existsSync(yamlPath)) {
    console.error(`❌ 文件不存在: ${yamlPath}`);
    process.exit(1);
}

const server = new SwaggerServer(port, title, yamlPath);
server.start();

// 优雅关闭
process.on('SIGINT', () => {
    server.stop();
    process.exit(0);
});

process.on('SIGTERM', () => {
    server.stop();
    process.exit(0);
});
