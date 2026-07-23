from __future__ import annotations

from typing import Any

import pytest

from mimir.utility import DatabaseClient


class _FakeCursor:
    def __init__(self) -> None:
        self.description: list[tuple[str]] = [("id",), ("name",)]
        self._rows: list[tuple[Any, ...]] = [(1, "alpha"), (2, "beta")]
        self.executed: list[tuple[str, tuple[Any, ...] | None]] = []
        self.executemany_calls: list[tuple[str, list[tuple[Any, ...]]]] = []

    def execute(self, query: str, params: tuple[Any, ...] | None = None) -> None:
        self.executed.append((query, params))

    def executemany(self, query: str, values: list[tuple[Any, ...]]) -> None:
        self.executemany_calls.append((query, values))

    def fetchall(self) -> list[tuple[Any, ...]]:
        return self._rows


class _FakeConnection:
    def __init__(self) -> None:
        self.cursor_instance = _FakeCursor()
        self.committed = False
        self.closed = False

    def cursor(self) -> _FakeCursor:
        return self.cursor_instance

    def commit(self) -> None:
        self.committed = True

    def close(self) -> None:
        self.closed = True


class _FakeUtility(DatabaseClient):
    def connect(self, auth_method: str = "password", **kwargs: Any) -> None:
        _ = auth_method
        _ = kwargs
        self._connection = _FakeConnection()


@pytest.mark.unit
def test_empty_instance_is_not_connected() -> None:
    utility = _FakeUtility()

    assert utility.is_connected is False


@pytest.mark.unit
def test_read_table_requires_connection() -> None:
    utility = _FakeUtility()

    with pytest.raises(RuntimeError, match="Call connect first"):
        utility.read_table("example_table")


@pytest.mark.unit
def test_write_table_requires_connection() -> None:
    utility = _FakeUtility()

    with pytest.raises(RuntimeError, match="Call connect first"):
        utility.write_table("example_table", [{"id": 1}])


@pytest.mark.unit
def test_read_table_returns_dict_rows() -> None:
    utility = _FakeUtility()
    utility.connect()

    rows = utility.read_table("sample.table", columns=["id", "name"], limit=10)

    assert rows == [{"id": 1, "name": "alpha"}, {"id": 2, "name": "beta"}]


@pytest.mark.unit
def test_write_table_appends_rows_and_commits() -> None:
    utility = _FakeUtility()
    utility.connect()

    row_count = utility.write_table("sample.table", [{"id": 1, "name": "alpha"}])

    connection = utility._require_connection()
    assert row_count == 1
    assert connection.committed is True
    assert connection.cursor_instance.executemany_calls == [
        ("INSERT INTO sample.table (id, name) VALUES (?, ?)", [(1, "alpha")])
    ]


@pytest.mark.unit
def test_write_table_replace_deletes_existing_rows() -> None:
    utility = _FakeUtility()
    utility.connect()

    utility.write_table("sample.table", [{"id": 1, "name": "alpha"}], if_exists="replace")

    connection = utility._require_connection()
    assert connection.cursor_instance.executed[0][0] == "DELETE FROM sample.table"


@pytest.mark.unit
def test_disconnect_closes_connection() -> None:
    utility = _FakeUtility()
    utility.connect()

    connection = utility._require_connection()
    utility.disconnect()

    assert connection.closed is True
    assert utility.is_connected is False
