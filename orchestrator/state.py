import os
import time
from typing import Any

import yaml

ALLOWED_STATUSES = frozenset({"pending", "in_progress", "completed", "failed"})


class State:
    def __init__(
        self,
        milestones: list[dict[str, Any]],
        path: str = "state.yaml",
    ):
        self._path = path
        self._data: dict[str, Any] = {"milestones": {}}
        self._init_from_milestones(milestones)
        self._load()

    def _init_from_milestones(self, milestones: list[dict[str, Any]]) -> None:
        for ms in milestones:
            name = ms.get("name")
            if not name:
                continue
            if name not in self._data["milestones"]:
                self._data["milestones"][name] = {
                    "status": "pending",
                    "timestamp": None,
                }

    def _load(self) -> None:
        if not os.path.exists(self._path):
            return
        try:
            with open(self._path) as f:
                data = yaml.safe_load(f)
        except Exception:
            return
        if not isinstance(data, dict) or "milestones" not in data:
            return
        for name, info in data["milestones"].items():
            if name in self._data["milestones"]:
                self._data["milestones"][name] = info

    def get(self, milestone: str) -> str:
        return self._data["milestones"][milestone]["status"]

    def set(self, milestone: str, status: str) -> None:
        if status not in ALLOWED_STATUSES:
            raise ValueError(
                f"Invalid status {status!r}. Must be one of {sorted(ALLOWED_STATUSES)}"
            )
        self._data["milestones"][milestone] = {
            "status": status,
            "timestamp": time.time(),
        }

    def get_all(self) -> dict[str, str]:
        return {
            name: info["status"]
            for name, info in self._data["milestones"].items()
        }

    def is_completed(self, milestone: str) -> bool:
        return self.get(milestone) == "completed"

    def all_completed(self) -> bool:
        return all(
            info["status"] == "completed"
            for info in self._data["milestones"].values()
        )

    def save(self, path: str | None = None) -> None:
        p = path or self._path
        with open(p, "w") as f:
            yaml.dump(self._data, f)
