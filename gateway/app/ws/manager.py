import asyncio
from collections import defaultdict
from fastapi import WebSocket


class ConnectionManager:
    """Manages WebSocket connections grouped by user ID."""

    def __init__(self):
        self._connections: dict[str, list[WebSocket]] = defaultdict(list)
        self._lock = asyncio.Lock()

    async def connect(self, user_id: str, ws: WebSocket):
        await ws.accept()
        async with self._lock:
            self._connections[user_id].append(ws)

    async def disconnect(self, user_id: str, ws: WebSocket):
        async with self._lock:
            conns = self._connections.get(user_id, [])
            if ws in conns:
                conns.remove(ws)
            if not conns:
                self._connections.pop(user_id, None)

    async def send_to_user(self, user_id: str, data: dict):
        async with self._lock:
            conns = list(self._connections.get(user_id, []))
        for ws in conns:
            try:
                await ws.send_json(data)
            except Exception:
                await self.disconnect(user_id, ws)

    def is_online(self, user_id: str) -> bool:
        return bool(self._connections.get(user_id))


# Shared instances
chat_manager = ConnectionManager()
notification_manager = ConnectionManager()
