#!/usr/bin/env python3
import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs, unquote, urlparse


SEEDS_DIR = Path(os.environ.get("MOCK_EXTERNAL_SEEDS_DIR", "/app/seeds"))


def load_json(name):
    with (SEEDS_DIR / name).open("r", encoding="utf-8") as handle:
        return json.load(handle)


DATA = {
    "bookings": load_json("demo-bookings.json").get("items", []),
    "workspace_bookings": load_json("demo-workspace-bookings.json").get("items", []),
    "payments": load_json("demo-payments.json").get("items", []),
    "users": load_json("demo-users.json").get("items", []),
    "knowledge": load_json("knowledge-base.json").get("articles", []),
    "providers": load_json("mock-external-services.json").get("providers", []),
}


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        parsed = urlparse(self.path)
        path = unquote(parsed.path)
        query = parse_qs(parsed.query)

        if path == "/health":
            return self.write_json({"status": "ok"})
        if path == "/api/v1/providers":
            return self.write_json({"providers": DATA["providers"]})
        if path.startswith("/api/v1/bookings/"):
            return self.find_by_identifier("bookings", "booking_number", path.rsplit("/", 1)[-1])
        if path == "/api/v1/bookings":
            return self.find_by_identifier("bookings", "identifiers", first(query.get("phone")))
        if path.startswith("/api/v1/workspace-bookings/"):
            return self.find_by_identifier("workspace_bookings", "booking_number", path.rsplit("/", 1)[-1])
        if path == "/api/v1/workspaces/availability":
            return self.workspace_availability(first(query.get("date")), first(query.get("type")))
        if path.startswith("/api/v1/payments/"):
            return self.find_by_identifier("payments", "payment_id", path.rsplit("/", 1)[-1])
        if path == "/api/v1/payments":
            return self.find_by_identifier("payments", "identifiers", first(query.get("order_id")))
        if path.startswith("/api/v1/accounts/by-phone/"):
            return self.find_by_identifier("users", "phone", path.rsplit("/", 1)[-1])
        if path.startswith("/api/v1/accounts/by-email/"):
            return self.find_by_identifier("users", "email", path.rsplit("/", 1)[-1])
        if path.startswith("/api/v1/accounts/"):
            return self.find_by_identifier("users", "user_id", path.rsplit("/", 1)[-1])
        if path in {"/api/v1/prices/services", "/api/v1/prices/workspaces", "/api/v1/rules/workspaces"}:
            return self.find_knowledge(path)

        return self.write_json({"error": "not_found", "path": path}, status=404)

    def find_by_identifier(self, collection, field, value):
        value = normalize(value)
        for item in DATA[collection]:
            candidate = item.get(field)
            if isinstance(candidate, list):
                matched = value in {normalize(part) for part in candidate}
            else:
                matched = normalize(candidate) == value
            if matched:
                payload = dict(item)
                payload["found"] = True
                payload["source"] = "mock_external"
                return self.write_json(payload)
        return self.write_json({"found": False, "source": "mock_external"}, status=404)

    def workspace_availability(self, date, workspace_type):
        for item in DATA["workspace_bookings"]:
            if item.get("date") == date and normalize(item.get("workspace_type")) == normalize(workspace_type):
                return self.write_json({
                    "found": True,
                    "workspace_type": item.get("workspace_type"),
                    "date": date,
                    "available": item.get("status") == "available",
                    "source": "mock_external",
                })
        return self.write_json({"found": False, "source": "mock_external"}, status=404)

    def find_knowledge(self, path):
        keys = {
            "/api/v1/prices/services": "services",
            "/api/v1/prices/workspaces": "workspace",
            "/api/v1/rules/workspaces": "workspace",
        }
        needle = keys[path]
        for article in DATA["knowledge"]:
            if needle in normalize(article.get("key")):
                payload = dict(article)
                payload["found"] = True
                payload["source"] = "mock_external"
                return self.write_json(payload)
        return self.write_json({"found": False, "source": "mock_external"}, status=404)

    def write_json(self, payload, status=200):
        data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, fmt, *args):
        return


def first(values):
    return values[0] if values else ""


def normalize(value):
    return str(value or "").strip().upper()


def main():
    host = os.environ.get("MOCK_EXTERNAL_HOST", "0.0.0.0")
    port = int(os.environ.get("MOCK_EXTERNAL_PORT", "8090"))
    ThreadingHTTPServer((host, port), Handler).serve_forever()


if __name__ == "__main__":
    main()
