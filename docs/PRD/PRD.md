# GitHub Personally Identifiable Data Scanner (PI Scanner) PRD (Multi-Layer, AU Banking)

### TL;DR

A CLI-first, multi-layer PI scanner purpose-built for Commonwealth Bank
source repositories. It combines Gitleaks and regex for broad PI
exposure detection, advances accuracy and context validation with ML/NLP
models (e.g., DeBERTa PII), and incorporates algorithmic verification
for Australian-specific identifiers (TFN, Medicare, etc.). Results are
context-scored for business and regulatory risk using AU banking
policies, flagging critical combinations (e.g., name + address + bank
info) and incorporating context-aware suppression for synthetic/test
data. CLI-first, focused on high-confidence detection and actionable
risk analysis before scaling to CI/CD integration.

------------------------------------------------------------------------

## Goals

### Business Goals

- Identify and quantify personally identifiable information (PII/PI)
  exposure across priority CBA code repositories while enabling
  compliance with AU banking and privacy regulations.

- Deliver highly accurate scanning, maximizing true positive detection
  and minimizing false positives to build trust and enable effective
  remediation.

- Support risk-based prioritization in reporting, aligned with AU
  banking risk ratings (Critical, High, Medium, Low).

- Support iterative improvement and adaptation as
  regulatory/organizational needs evolve.

### User Goals

- Security/platform teams can run scans on demand, receive risk-ranked,
  context-rich results, and rapidly validate findings.

- Easy suppression/configuration for known-safe synthetic/test data and
  tuning of PI types/patterns.

### Non-Goals (Initial Phase)

- Automated CICD pipeline or pre-commit integration (CLI only).

- Automated remediation—detection, contextual risk scoring, and
  reporting only.

- Coverage outside CBA GitHub repositories.

------------------------------------------------------------------------

## User Stories

**Personas:**

- Security Engineer: Runs manual CLI scans, receives contextual,
  risk-ranked results mapped to AU regulatory terminology, and
  prioritizes critical findings for triage.

- Platform Engineer: Configures scanning (target repos, PI types,
  context rules), optimizes for accuracy, tunes ignores/suppressions,
  and prepares for future integration.

**Sample Stories:**

- As a Security Engineer, I want to scan a prioritized repo list, so I
  can surface and report critical real PI exposures in line with AU
  regulatory risk.

- As a Platform Engineer, I want to tune context scoring and suppression
  for synthetic/test data, so scanning output reflects true disclosure
  risk, not noise.

- As a Security Engineer, I want to cross-validate regex/Gitleaks
  findings using ML/NLP models, so I can have high trust in reported PI
  exposures before wider rollout.

------------------------------------------------------------------------

## Functional Requirements

- **Detection Architecture (Priority: High)**

  - Multi-stage pipeline:

    - **Stage 1:** Gitleaks + custom regex for PI type detection (broad
      net, configurable).

    - **Stage 2:** ML/NLP model (e.g., DeBERTa PII) for
      semantic/contextual validation, reducing false positives.

    - **Stage 3:** Algorithmic validation for Australian-specific
      entities (TFN, ABN, Medicare, Driver's License, etc.).

  - Configurable PI type inclusion—must include names, addresses,
    phones, emails, accounts, health/financial data, IPs, and all
    AU-specific regulatory variants.

  - Contextual risk scoring based on:

    - Entity co-occurrence (e.g., name + address + TFN = “Critical”
      risk)

    - Environment awareness (e.g., production vs. test/synthetic data,
      flagged via file path/structure or metadata)

  - Data/test awareness: Intelligent flagging and potential suppression
    of likely synthetic/test/mocked data via configuration (path-based,
    pattern-based).

- **Reporting (Priority: High)**

  - Findings are risk-ranked per AU banking regulatory standards
    (Critical, High, Medium, Low).

  - Supports SARIF, JSON, and CSV outputs suitable for
    audit/tracking/review.

  - CLI flags for custom rule sets, ignore lists, suppression
    mechanisms, and output destinations.

- **Workflow & Tuning (Priority: High)**

  - CLI accepts list of repos (local or remote), processes each via full
    detection pipeline, generates output artifacts.

  - Full configuration support for:

    - Ignore regex/patterns

    - File/path suppression

    - PI type/risk context tuning

    - Context scoring and suppression rules

- **Performance (Medium)**

  - Supports parallel repo/file scanning.

  - Optimized for large, data-rich repositories (includes incremental
    scan, hash-based caching where possible).

  - Robust CLI error/status reporting for scan completion, failures, and
    configuration issues.

- **Future/Out-of-Scope for Initial Release**

  - Automated CI/CD, pre-commit, or PR integration.

  - Automated code remediation or PI data redaction.

  - Direct developer feedback/notifications.

------------------------------------------------------------------------

## User Experience

**Entry Point & First-Time User Experience**

- User installs/configures the CLI.

- User prepares a config specifying target repositories, PI types of
  interest, and any ignore/suppression patterns (default/guided
  templates provided).

**Core Experience**

1.  **Step 1: Configure and Initiate Scan**

    - User provides list of local or remote repos and initiates scan
      using CLI command and configuration file/flags.

    - CLI verifies access, validates configuration, and reports
      readiness.

2.  **Step 2: Multi-Layer Detection and Scoring**

    - Scanner applies Gitleaks/regex, pipes matches to ML model for
      semantic validation, and applies AU-specific validation.

    - Results are contextually scored:

      - Standalone names, emails, or benign records: “Low Risk”

      - Verified sensitive combinations (e.g., name + TFN + address):
        “Critical Risk” per AU policy

3.  **Step 3: Risk-Ranked Output**

    - Output files (SARIF, CSV, JSON) are generated with risk-rank,
      entity breakdown, and explanatory context/rule logic.

    - Each finding includes file, exact match, risk rationale, and
      suppression info (if flagged as test/synthetic).

4.  **Step 4: Iterative Review and Tuning**

    - User reviews output. Suppresses, tunes, or updates ignore patterns
      for false positives or test/synthetic data.

    - User may rerun in “dry run” or incremental mode for fast
      retesting.

**Advanced Features & Edge Cases**

- Drill-down to review the justification for each risk score, including
  which PI types co-occur, ML model scoring, AU validator results, and
  any suppression chains.

- Support for “diff only” scans for efficiency.

- Handling of edge cases: large files, malformed data, mixed encodings.

**UI/UX Highlights**

- Output files are legible and actionable—risk is prominent, with
  context and paths shown.

- Export is directly suitable for further triage/import into Excel,
  Splunk, or bank security dashboards.

- CLI includes detailed help, guided error explanations, and sample
  templates for easy onboarding.

------------------------------------------------------------------------

## Narrative

Sam, a security engineer at Commonwealth Bank, is tasked with ensuring
no accidental PI exposure leaks into mission-critical data pipelines. He
configures the PI scanner for comprehensive AU compliance, including
TFNs, Medicare, and financial record layouts. Running the CLI scan
across ten prioritized repositories, Sam gets a short list of findings:
each is clearly ranked—“Critical,” “High,” “Medium,” or “Low”—with
evidence and rationale.  
One flagged file exposes a full name, physical address, and ABN number
in a configuration script; the scanner rates this as “Critical” with
supporting context under AU disclosure policies. Meanwhile, a synthetic
test dataset in /mock/ is labeled “No Risk” and suppressed automatically
thanks to tuned config patterns.  
With each scan, Sam quickly validates real findings, refines the
ignore/suppression list, and tunes the configuration to adapt as new
data types emerge. This high-trust, actionable process builds
confidence, demonstrating to both the security team and auditors that
CBA’s repositories are under rigorous, regulation-grade PI surveillance.

------------------------------------------------------------------------

## Success Metrics

<table style="min-width: 75px">
<tbody>
<tr>
<th><p>Metric</p></th>
<th><p>Target/Measurement</p></th>
<th><p>Source/Tracking</p></th>
</tr>
&#10;<tr>
<td><p>Precision/Recall</p></td>
<td><p>&gt;95% accuracy; &lt;5% false positives/negatives</p></td>
<td><p>Manual review of pilot results</p></td>
</tr>
<tr>
<td><p>Pilot Coverage</p></td>
<td><p>100% of 10 pilot repos scanned &amp; results delivered</p></td>
<td><p>CLI scan usage logs</p></td>
</tr>
<tr>
<td><p>Remediation Rate</p></td>
<td><p>Closure of all “Critical/High” findings post-scan</p></td>
<td><p>Issue tracker, remediation log</p></td>
</tr>
<tr>
<td><p>Risk Suppression Accuracy</p></td>
<td><p>100% correct suppression of test/synthetic data</p></td>
<td><p>User audit, sampled validation</p></td>
</tr>
<tr>
<td><p>Iterative Tuning Count</p></td>
<td><p>&lt;3 config iteration cycles to reach target accuracy</p></td>
<td><p>Usage logs, user feedback</p></td>
</tr>
<tr>
<td><p>Reporting/Audit Fit</p></td>
<td><p>Positive user/advisor feedback on risk reporting format</p></td>
<td><p>Survey/interview/feedback</p></td>
</tr>
</tbody>
</table>

### Tracking Plan

- CLI run events and parameters (repo, config, duration)

- Number, type, and score of findings per repo/scan

- False positive suppressions and tuning actions

- Timing and throughput for scan completion

------------------------------------------------------------------------

## Technical Considerations

### Technical Needs

- CLI packaging (cross-platform compatibility, dependency management)

- Core engine orchestration (multi-stage: regex → ML/NLP → AU validator)

- Secure output writing and deletion logic

### Integration Points

- ML model hosting (local or via secure on-prem/offline cloud instance)

- Support for Gitleaks, Regex PI libraries, custom AU validator modules

- Onboarding for AU-specific policy/credential validation logic

### Data Storage & Privacy

- All output artifacts handled as PI data; secure storage and access

- Integration with CBA’s internal secure file-handling SOPs

### Scalability & Performance

- Parallel processing for multi-repo scanning

- Incremental scan/caching for large codebases

### Potential Challenges

- Managing ML/NLP model hosting and updates

- Performance on very large, data-heavy repos

- False positive reduction without missing newly added PI types

------------------------------------------------------------------------

## Milestones & Sequencing

### Project Estimate

- **Medium Project:** 4 weeks

### Team Size & Composition

- **Lean Team:** 2 engineers (ML + software), 1 security lead, 1
  product/PM (with strong AU PI and banking regulatory context)

### Suggested Phases

**Phase 1: POC CLI Build (Week 1)**

- Key Deliverables: Engine with baseline Gitleaks/regex, basic AU
  validator; CLI runs and generates basic output

- Dependencies: Gitleaks, existing AU policy specs

**Phase 2: ML/NLP & Contextual Scoring (Week 2)**

- Key Deliverables: ML/NLP integration (e.g., DeBERTa PII); risk logic
  and score mapping; initial suppression/tuning config

- Dependencies: Access to ML models, CBA regulatory rules

**Phase 3: User Testing & Tuning (Week 3)**

- Key Deliverables: Full parallel scan on 10 pilot repos; config UI
  improvements; robust output; risk suppression logic; team validation

- Dependencies: Pilot repo list, security user involvement

**Phase 4: Reporting, Documentation, Feedback (Week 4)**

- Key Deliverables: Comprehensive audit-grade reporting
  (SARIF/CSV/JSON); end-user and maintainer guides; collect/implement
  feedback, define readiness for next phase (CI/CD integration)

- Dependencies: Pilot feedback, user testing sessions

------------------------------------------------------------------------
