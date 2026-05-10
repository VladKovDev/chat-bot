#!/usr/bin/env python3
"""
Simple console chat adapter for testing the decision-engine service.
"""

import requests
from uuid import uuid4


class ConsoleChat:
    """Console chat client for decision-engine."""

    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        self.sessions_endpoint = f"{base_url}/api/v1/sessions"
        self.messages_endpoint = f"{base_url}/api/v1/messages"
        self.channel = "dev-cli"
        self.client_id = f"console-{uuid4()}"
        self.session_id: str | None = None
        self.message_number = 0

    def ensure_session(self) -> str:
        """Start or resume a decision-engine session for this console client."""
        if self.session_id:
            return self.session_id

        payload = {
            "channel": self.channel,
            "client_id": self.client_id,
        }
        response = requests.post(
            self.sessions_endpoint,
            json=payload,
            headers={"Content-Type": "application/json"},
            timeout=10,
        )
        response.raise_for_status()
        body = response.json()
        self.session_id = body["session_id"]
        return self.session_id

    def send_message(self, text: str) -> dict:
        """Send message to decision-engine and return response."""
        try:
            session_id = self.ensure_session()
            self.message_number += 1
            payload = {
                "session_id": session_id,
                "channel": self.channel,
                "external_message_id": f"{self.client_id}-{self.message_number}",
                "text": text,
            }
            response = requests.post(
                self.messages_endpoint,
                json=payload,
                headers={"Content-Type": "application/json"},
                timeout=10
            )
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            return {
                "error": {
                    "message": f"Connection error: {e}",
                },
                "text": "",
                "mode": "",
            }

    def print_response(self, response: dict):
        """Pretty print the response from decision-engine."""
        if not response.get("error"):
            print(f"\n🤖 Bot: {response.get('text', '')}")
            if response.get("mode"):
                print(f"   Mode: {response.get('mode')}")
            if response.get("quick_replies"):
                print("   Quick replies:")
                for i, option in enumerate(response.get("quick_replies", []), 1):
                    print(f"      {i}. {option.get('label')}")
        else:
            error = response.get("error") or {}
            print(f"\n❌ Error: {error.get('message', 'Unknown error')}")
        print()

    def run(self):
        """Run the console chat loop."""
        print("=== Console Chat Adapter ===")
        print(f"Connected to: {self.messages_endpoint}")
        print("Type 'quit' or 'exit' to stop the chat")
        print("Type 'session' to start a new console session")
        print("-" * 40)

        while True:
            try:
                label = self.session_id[:8] if self.session_id else "new"
                user_input = input(f"[Session {label}] You: ").strip()

                if not user_input:
                    continue

                # Handle special commands
                if user_input.lower() in ["quit", "exit"]:
                    print("Goodbye!")
                    break

                if user_input.lower() == "session":
                    self.client_id = f"console-{uuid4()}"
                    self.session_id = None
                    self.message_number = 0
                    print("✓ New console session will be created on next message\n")
                    continue

                # Send message and display response
                response = self.send_message(user_input)
                self.print_response(response)

            except KeyboardInterrupt:
                print("\n\nGoodbye!")
                break
            except Exception as e:
                print(f"\n❌ Unexpected error: {e}\n")


if __name__ == "__main__":
    chat = ConsoleChat()
    chat.run()
