import json
import uuid

import asyncpg
import httpx
from fastapi import APIRouter, Depends, WebSocket, WebSocketDisconnect

from app.auth import decode_token
from app.config import settings
from app.database import get_pool
from app.schemas.envelope import ok
from app.ws.manager import chat_manager, notification_manager
from app.ws.chat_relay import moderate_message

router = APIRouter(tags=["chat"])


@router.websocket("/ws/chat")
async def chat_ws(ws: WebSocket):
    """WebSocket endpoint for real-time chat.

    Client must send an initial auth message: {"token": "jwt_access_token"}
    Then send messages: {"conversation_id": "...", "content": "..."}
    """
    await ws.accept()

    # Authenticate
    try:
        auth_msg = await ws.receive_json()
        token = auth_msg.get("token", "")
        payload = decode_token(token)
        if payload.get("type") != "access":
            await ws.close(code=4001, reason="Invalid token type")
            return
        user_id = payload["sub"]
    except Exception:
        await ws.close(code=4001, reason="Authentication failed")
        return

    # Register connection
    await chat_manager.connect(user_id, ws)
    await ws.send_json({"type": "connected", "user_id": user_id})

    pool = get_pool()

    try:
        while True:
            data = await ws.receive_json()
            conv_id = data.get("conversation_id")
            content = data.get("content", "").strip()

            if not conv_id or not content:
                await ws.send_json({"type": "error", "message": "conversation_id and content required"})
                continue

            async with pool.acquire() as conn:
                # Verify participant
                conv = await conn.fetchrow(
                    "SELECT participant_a, participant_b FROM conversations WHERE id = $1",
                    uuid.UUID(conv_id),
                )
                if not conv:
                    await ws.send_json({"type": "error", "message": "conversation not found"})
                    continue

                uid = uuid.UUID(user_id)
                if uid != conv["participant_a"] and uid != conv["participant_b"]:
                    await ws.send_json({"type": "error", "message": "not a participant"})
                    continue

                # Determine recipient
                recipient_id = str(conv["participant_b"] if uid == conv["participant_a"] else conv["participant_a"])

                # Moderate via xo
                mod_result = await moderate_message(content)
                mod_data = mod_result.get("data", {})

                if mod_data.get("status") == "blocked":
                    await ws.send_json({
                        "type": "message_blocked",
                        "reason": mod_data.get("reason", "Message blocked by moderation"),
                    })
                    continue

                sanitized = mod_data.get("sanitized", content)
                flags = mod_data.get("flags", {})
                status = mod_data.get("status", "clean")

                # Persist
                msg_id = await conn.fetchval(
                    """INSERT INTO chat_messages
                       (conversation_id, sender_id, content, content_moderated, moderation_flags, moderation_status)
                       VALUES ($1, $2, $3, $4, $5, $6)
                       RETURNING id""",
                    uuid.UUID(conv_id), uid, content, sanitized,
                    json.dumps(flags), status,
                )

                await conn.execute(
                    "UPDATE conversations SET updated_at = NOW() WHERE id = $1",
                    uuid.UUID(conv_id),
                )

                # Build outgoing message (uses sanitized content)
                out_msg = {
                    "type": "chat_message",
                    "id": str(msg_id),
                    "conversation_id": conv_id,
                    "sender_id": user_id,
                    "content": sanitized,
                    "moderation_status": status,
                    "created_at": str(await conn.fetchval("SELECT NOW()")),
                }

                # Send to sender (confirmation)
                await ws.send_json(out_msg)

                # Send to recipient if online
                if chat_manager.is_online(recipient_id):
                    await chat_manager.send_to_user(recipient_id, out_msg)
                else:
                    # Create in-app notification for offline recipient
                    await conn.execute(
                        """INSERT INTO inapp_notifications (user_id, type, title, body, payload)
                           VALUES ($1, 'chat_message', 'New message', $2, $3)""",
                        uuid.UUID(recipient_id),
                        f"New message in conversation",
                        json.dumps({"conversation_id": conv_id, "sender_id": user_id}),
                    )
                    # Push via notification WS if connected
                    await notification_manager.send_to_user(recipient_id, {
                        "type": "notification",
                        "notification_type": "chat_message",
                        "conversation_id": conv_id,
                    })

    except WebSocketDisconnect:
        pass
    finally:
        await chat_manager.disconnect(user_id, ws)


@router.websocket("/ws/notifications")
async def notifications_ws(ws: WebSocket):
    """WebSocket for real-time notification delivery."""
    await ws.accept()

    try:
        auth_msg = await ws.receive_json()
        token = auth_msg.get("token", "")
        payload = decode_token(token)
        if payload.get("type") != "access":
            await ws.close(code=4001, reason="Invalid token type")
            return
        user_id = payload["sub"]
    except Exception:
        await ws.close(code=4001, reason="Authentication failed")
        return

    await notification_manager.connect(user_id, ws)
    await ws.send_json({"type": "connected", "user_id": user_id})

    try:
        while True:
            # Keep connection alive, client can send pings
            await ws.receive_text()
    except WebSocketDisconnect:
        pass
    finally:
        await notification_manager.disconnect(user_id, ws)
