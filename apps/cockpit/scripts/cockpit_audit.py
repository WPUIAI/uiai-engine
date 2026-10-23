"""Versioned Cockpit audit-row contract shared by producers and audit-schema.py."""
SCHEMA = "uaiengine.cockpit.audit.v1"
REQUIRED = {"schema", "kind", "workflow", "run_id", "at"}
KINDS = {"cockpit_ci_failure", "self_heal_no_op", "self_heal_no_match", "self_heal_no_change", "self_heal_applied"}


def validate_row(row):
    if row.get("schema") != SCHEMA:
        return [f"expected schema {SCHEMA}"]
    missing = sorted(REQUIRED - row.keys())
    if missing:
        return [f"missing Cockpit audit fields {missing}"]
    if any(not isinstance(row[field], str) or not row[field] for field in REQUIRED):
        return ["Cockpit audit identity fields must be nonempty strings"]
    if row["kind"] not in KINDS:
        return ["unsupported Cockpit audit kind"]
    return []
