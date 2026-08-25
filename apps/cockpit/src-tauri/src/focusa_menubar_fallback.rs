//! UIAI-COCKPIT-005 T005-06.05 — Denial / mismatch / absence fallback (Path B → Path A).
//! Never mutates authority or creates partial profile on fallback; Path A stays available.

use serde::Serialize;

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
pub struct FallbackDecision {
    pub schema: &'static str,
    pub fallback_to_path_a: bool,
    pub reason: &'static str,
    pub authority_mutated: bool,
    pub partial_profile_created: bool,
}

pub fn decide_fallback(reason: &'static str) -> FallbackDecision {
    FallbackDecision {
        schema: "focusa.menubar_fallback_decision.v1",
        fallback_to_path_a: true,
        reason,
        authority_mutated: false,
        partial_profile_created: false,
    }
}

pub fn fallback_for_eligibility(eligible: bool, reason_str: &str) -> Option<FallbackDecision> {
    if eligible { return None; }
    let reason: &'static str = match reason_str {
        "different_machine" | "different_user" => "absent",
        "policy_disabled" => "denied_by_policy",
        "dismissed" => "denied_by_operator",
        "daemon_mismatch" => "mismatch",
        _ => "absent",
    };
    Some(decide_fallback(reason))
}

pub fn fallback_for_mismatch() -> FallbackDecision { decide_fallback("mismatch") }
pub fn fallback_for_absence() -> FallbackDecision { decide_fallback("absent") }

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn no_mutation_on_fallback() {
        for r in ["mismatch","absent","denied_by_policy","denied_by_operator"] {
            let d = decide_fallback(r);
            assert!(d.fallback_to_path_a);
            assert!(!d.authority_mutated);
            assert!(!d.partial_profile_created);
            assert_eq!(d.schema, "focusa.menubar_fallback_decision.v1");
        }
    }
    #[test]
    fn eligibility_maps_to_fallback() {
        assert_eq!(fallback_for_eligibility(false, "policy_disabled").unwrap().reason, "denied_by_policy");
        assert!(fallback_for_eligibility(true, "eligible").is_none());
    }
}
