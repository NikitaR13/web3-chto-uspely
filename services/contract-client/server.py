# services/contract-client/server.py
import os
import json
import requests
from flask import Flask, request, jsonify

app = Flask(__name__)

NEO_RPC = os.getenv("NEO_RPC", "http://neo-node:10332")
CONTRACT_HASH = os.getenv("CONTRACT_HASH", "0x1234abcd...")

@app.route("/has_access", methods=["GET"])
def has_access():
    user = request.args.get("user", "")
    content_id = request.args.get("content_id", "")

    body = {
        "jsonrpc": "2.0",
        "method": "invokefunction",
        "params": [
            CONTRACT_HASH,
            "has_access",
            [
                {"type": "Hash160", "value": user},
                {"type": "String",  "value": content_id}
            ]
        ],
        "id": 1
    }
    resp = requests.post(NEO_RPC, json=body).json()
    stack = resp.get("result", {}).get("stack", [])
    result = False
    if stack and stack[0]["type"] == "Boolean":
        result = stack[0]["value"]
    return jsonify({"has": result})

@app.route("/purchase_access", methods=["POST"])
def purchase_access():
    """
    В реальности нужно:
    1) user -> подписывает transaction transfer(GAS) -> contractHash, amount=1, data=content_id
    2) onNEP17Payment -> вызывает purchase_access внутри контракта.

    Здесь упрощённая демо: вызываем invokefunction (что не обеспечивает реальный перевод GAS).
    """
    data = request.get_json()
    if not data:
        return jsonify({"error": "Invalid JSON"}), 400
    user = data.get("user", "")
    content_id = data.get("content_id", "")

    # !!! Это не настоящая оплата. Просто вызываем purchase_access, БЕЗ реального transfer.
    body = {
        "jsonrpc": "2.0",
        "method": "invokefunction",
        "params": [
            CONTRACT_HASH,
            "purchase_access",
            [
                {"type": "Hash160", "value": user},
                {"type": "String",  "value": content_id}
            ]
        ],
        "id": 1
    }
    resp = requests.post(NEO_RPC, json=body).json()
    return jsonify(resp)

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5001, debug=False)
