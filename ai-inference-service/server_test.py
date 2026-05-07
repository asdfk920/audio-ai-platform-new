"""
BSRoformer SCNet 音轨分离服务 - 简化测试版
"""

from http.server import HTTPServer, BaseHTTPRequestHandler
import json
from urllib.parse import urlparse


class BSRoformerHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        parsed_path = urlparse(self.path)
        
        if parsed_path.path == "/health":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.end_headers()
            response = {
                "status": "healthy",
                "model_name": "bsroformer_scnet",
                "device": "cpu",
                "gpu_available": False
            }
            self.wfile.write(json.dumps(response).encode())
            print(f"[INFO] GET /health - 200 OK")
            
        elif parsed_path.path == "/":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.end_headers()
            response = {
                "service": "BSRoformer SCNet Audio Separation Service",
                "version": "1.0.0",
                "status": "running"
            }
            self.wfile.write(json.dumps(response).encode())
            print(f"[INFO] GET / - 200 OK")
            
        else:
            self.send_response(404)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            error = {"error": "Not Found", "path": parsed_path.path}
            self.wfile.write(json.dumps(error).encode())
            print(f"[WARN] GET {parsed_path.path} - 404 Not Found")
    
    def do_POST(self):
        parsed_path = urlparse(self.path)
        
        if parsed_path.path == "/api/v1/separate":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.end_headers()
            response = {
                "success": True,
                "message": "音轨分离成功（演示模式）",
                "output_files": {
                    "vocals": "/app/data/output/demo_vocals.wav",
                    "instrumental": "/app/data/output/demo_instrumental.wav"
                },
                "processing_time": 0.5
            }
            self.wfile.write(json.dumps(response).encode())
            print(f"[INFO] POST /api/v1/separate - 200 OK")
        else:
            self.send_response(404)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            error = {"error": "Not Found"}
            self.wfile.write(json.dumps(error).encode())
            print(f"[WARN] POST {parsed_path.path} - 404 Not Found")
    
    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()
        print(f"[INFO] OPTIONS - 200 OK")
    
    def log_message(self, format, *args):
        print(f"[{self.log_date_time_string()}] {format % args}")


def main():
    host = "0.0.0.0"
    port = 8004
    
    server = HTTPServer((host, port), BSRoformerHandler)
    print("=" * 60)
    print("BSRoformer SCNet 音轨分离服务 - 简化测试版")
    print("=" * 60)
    print(f"服务器地址：http://{host}:{port}")
    print(f"健康检查：http://localhost:{port}/health")
    print(f"音轨分离：POST http://localhost:{port}/api/v1/separate")
    print("=" * 60)
    print("按 Ctrl+C 停止服务")
    print()
    
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n正在关闭服务...")
        server.shutdown()
        print("服务已停止")


if __name__ == "__main__":
    main()
