"""
简化的 BSRoformer 测试服务器
不依赖完整的环境，仅用于测试
"""

from http.server import HTTPServer, BaseHTTPRequestHandler
import json


class SimpleHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            response = {
                "status": "healthy",
                "model_name": "bsroformer_scnet",
                "device": "cpu",
                "gpu_available": False
            }
            self.wfile.write(json.dumps(response).encode())
        elif self.path == "/":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            response = {
                "service": "BSRoformer SCNet Audio Separation Service",
                "version": "1.0.0",
                "status": "running (simplified mode)"
            }
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        if self.path == "/api/v1/separate":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
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
        else:
            self.send_response(404)
            self.end_headers()
    
    def log_message(self, format, *args):
        print(f"[{self.log_date_time_string()}] {format % args}")


def run_server(port=8004):
    server_address = ("", port)
    httpd = HTTPServer(server_address, SimpleHandler)
    print(f"Starting server on port {port}...")
    print(f"Health check: http://localhost:{port}/health")
    print(f"API docs: http://localhost:{port}/docs")
    print("Press Ctrl+C to stop")
    httpd.serve_forever()


if __name__ == "__main__":
    run_server()
