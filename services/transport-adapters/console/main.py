#!/usr/bin/env python3
"""
Simple console chat adapter for testing the decision-engine service.
"""

import requests
import json


class ConsoleChat:
    """Console chat client for decision-engine."""

    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        self.endpoint = f"{base_url}/api/v1/messages"
        self.chat_id = 1

    def send_message(self, text: str) -> dict:
        """Send message to decision-engine and return response."""
        payload = {
            "text": text,
            "type": "user_message",
            "channel": "dev-cli",
            "chat_id": self.chat_id
        }

        try:
            response = requests.post(
                self.endpoint,
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
        print(f"Connected to: {self.endpoint}")
        print("Type 'quit' or 'exit' to stop the chat")
        print("Type 'chat <id>' to change chat ID")
        print("-" * 40)

        while True:
            try:
                user_input = input(f"[Chat #{self.chat_id}] You: ").strip()

                if not user_input:
                    continue

                # Handle special commands
                if user_input.lower() in ["quit", "exit"]:
                    print("Goodbye!")
                    break

                if user_input.lower().startswith("chat "):
                    try:
                        new_id = int(user_input.split()[1])
                        self.chat_id = new_id
                        print(f"✓ Chat ID changed to {new_id}\n")
                        continue
                    except (ValueError, IndexError):
                        print("❌ Invalid chat ID. Usage: chat <number>\n")
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
