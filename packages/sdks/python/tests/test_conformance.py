import json
import unittest
from pathlib import Path
from typing import Any

from aamp_sdk import parse_aamp_headers


FIXTURE_PATH = (
    Path(__file__).resolve().parents[4]
    / "conformance"
    / "fixtures"
    / "parser.json"
)


class ConformanceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.fixtures: dict[str, Any] = json.loads(FIXTURE_PATH.read_text(encoding="utf-8"))

    def test_shared_parser_fixtures(self) -> None:
        for fixture in self.fixtures["cases"]:
            with self.subTest(fixture=fixture["name"]):
                parsed = parse_aamp_headers(fixture["input"])
                self.assertIsNotNone(parsed)
                assert parsed is not None

                expected = fixture["expected"]
                actual = {key: parsed.get(key) for key in expected}
                self.assertEqual(actual, expected)


if __name__ == "__main__":
    unittest.main()
