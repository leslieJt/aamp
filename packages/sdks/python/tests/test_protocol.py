import unittest

from aamp_sdk import (
    build_dispatch_headers,
    build_pair_request_headers,
    build_pair_respond_headers,
    build_result_headers,
    parse_aamp_headers,
    parse_dispatch_context_header,
    serialize_dispatch_context_header,
)


class ProtocolTests(unittest.TestCase):
    def test_dispatch_context_round_trip(self) -> None:
        value = serialize_dispatch_context_header({"project_key": "proj 123", "user_key": "alice"})
        self.assertEqual(value, "project_key=proj%20123; user_key=alice")
        self.assertEqual(
            parse_dispatch_context_header(value),
            {"project_key": "proj 123", "user_key": "alice"},
        )

    def test_build_dispatch_headers(self) -> None:
        headers = build_dispatch_headers(
            "task-1",
            priority="urgent",
            dispatch_context={"project_key": "proj-1"},
        )
        self.assertEqual(headers["X-AAMP-Intent"], "task.dispatch")
        self.assertEqual(headers["X-AAMP-TaskId"], "task-1")

    def test_dispatch_session_key_round_trip(self) -> None:
        headers = build_dispatch_headers("task-1", session_key="sess-1")
        self.assertEqual(headers["X-AAMP-Session-Key"], "sess-1")
        parsed = parse_aamp_headers(
            {
                "from": "dispatcher@example.com",
                "to": "agent@example.com",
                "subject": "[AAMP Task] Do something",
                "messageId": "<msg-1@example.com>",
                "bodyText": "Please do the work.",
                "headers": headers,
            }
        )
        self.assertIsNotNone(parsed)
        assert parsed is not None
        self.assertEqual(parsed["intent"], "task.dispatch")
        self.assertEqual(parsed["sessionKey"], "sess-1")

    def test_dispatch_session_key_empty(self) -> None:
        headers = build_dispatch_headers("task-1", session_key="")
        self.assertNotIn("X-AAMP-Session-Key", headers)
        parsed = parse_aamp_headers(
            {
                "from": "dispatcher@example.com",
                "to": "agent@example.com",
                "subject": "[AAMP Task] Do something",
                "headers": headers,
            }
        )
        self.assertIsNotNone(parsed)
        assert parsed is not None
        self.assertIsNone(parsed["sessionKey"])

    def test_dispatch_session_key_trimmed(self) -> None:
        headers = build_dispatch_headers("task-1", session_key="  sess-trim  ")
        self.assertEqual(headers["X-AAMP-Session-Key"], "sess-trim")

    def test_parse_task_result(self) -> None:
        headers = build_result_headers(
            "task-2",
            status="completed",
            output="done",
            structured_result=[{"fieldKey": "summary", "value": "done"}],
        )
        parsed = parse_aamp_headers(
            {
                "from": "agent@example.com",
                "to": "dispatcher@example.com",
                "subject": "[AAMP Result] Task task-2 - completed",
                "messageId": "<msg-2@example.com>",
                "bodyText": "Output:\ndone",
                "headers": headers,
            }
        )
        self.assertIsNotNone(parsed)
        assert parsed is not None
        self.assertEqual(parsed["intent"], "task.result")
        self.assertEqual(parsed["output"], "done")
        self.assertEqual(parsed["structuredResult"][0]["fieldKey"], "summary")

    def test_pair_request_round_trip(self) -> None:
        rules = {"source": ["feishu", "github"]}
        headers = build_pair_request_headers("pair-1", "abc123", rules)
        self.assertEqual(headers["X-AAMP-Intent"], "pair.request")
        self.assertEqual(headers["X-AAMP-Pair-Code"], "abc123")
        self.assertTrue(headers["X-AAMP-Dispatch-Context-Rules"])

        parsed = parse_aamp_headers(
            {
                "from": "bridge@example.com",
                "to": "agent@example.com",
                "messageId": "<pair-req-1@example.com>",
                "subject": "[AAMP Pair] Connection request",
                "bodyText": "AAMP Pair Request",
                "headers": headers,
            }
        )
        self.assertIsNotNone(parsed)
        assert parsed is not None
        self.assertEqual(parsed["intent"], "pair.request")
        self.assertEqual(parsed["taskId"], "pair-1")
        self.assertEqual(parsed["pairCode"], "abc123")
        self.assertEqual(parsed["dispatchContextRules"]["source"], ["feishu", "github"])

    def test_pair_request_missing_code(self) -> None:
        headers = build_pair_request_headers("pair-missing", "code", None)
        del headers["X-AAMP-Pair-Code"]
        parsed = parse_aamp_headers(
            {
                "from": "bridge@example.com",
                "to": "agent@example.com",
                "subject": "[AAMP Pair] Connection request",
                "headers": headers,
            }
        )
        self.assertIsNone(parsed)

    def test_pair_respond_failure_round_trip(self) -> None:
        headers = build_pair_respond_headers(
            "pair-rt-1",
            success=False,
            reason="invalid or expired pair code",
        )
        self.assertEqual(headers["X-AAMP-Status"], "rejected")
        self.assertEqual(headers["X-AAMP-ErrorMsg"], "invalid or expired pair code")

        parsed = parse_aamp_headers(
            {
                "from": "agent@example.com",
                "to": "app@example.com",
                "messageId": "<msg-pair-1@example.com>",
                "subject": "[AAMP Pair] rejected",
                "bodyText": "AAMP Pair Response",
                "headers": headers,
            }
        )
        self.assertIsNotNone(parsed)
        assert parsed is not None
        self.assertEqual(parsed["intent"], "pair.respond")
        self.assertEqual(parsed["taskId"], "pair-rt-1")
        self.assertFalse(parsed["success"])
        self.assertEqual(parsed["status"], "rejected")
        self.assertEqual(parsed["errorMsg"], "invalid or expired pair code")

    def test_pair_respond_success_omits_error_msg(self) -> None:
        headers = build_pair_respond_headers("pair-ok", success=True, reason="should be ignored")
        self.assertEqual(headers["X-AAMP-Status"], "completed")
        self.assertNotIn("X-AAMP-ErrorMsg", headers)

        parsed = parse_aamp_headers(
            {
                "from": "agent@example.com",
                "to": "app@example.com",
                "subject": "[AAMP Pair] completed",
                "headers": headers,
            }
        )
        self.assertIsNotNone(parsed)
        assert parsed is not None
        self.assertTrue(parsed["success"])
        self.assertEqual(parsed["status"], "completed")

    def test_validate_pair_code_rejects_header_injection(self) -> None:
        malicious = "abc\r\nX-AAMP-Intent: task.dispatch"
        with self.assertRaises(ValueError):
            build_pair_request_headers("pair-1", malicious, None)

    def test_validate_pair_code_accepts_base64url(self) -> None:
        for code in ("abc123", "YWJjMTIz", "YWJjMTIz-_"):
            headers = build_pair_request_headers("pair-1", code, None)
            self.assertEqual(headers["X-AAMP-Pair-Code"], code.strip())

    def test_validate_pair_code_rejects_invalid_token(self) -> None:
        for code in ("", "has space", "bad+plus", "bad/slash"):
            with self.assertRaises(ValueError):
                build_pair_request_headers("pair-1", code, None)


if __name__ == "__main__":
    unittest.main()
