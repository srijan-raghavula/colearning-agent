flowchart TD
    START([Start: Access Platform]) --> AUTH{Authenticated?}
    AUTH -->|No| LOGIN[Sign Up / Login] --> ROLE
    AUTH -->|Yes| ROLE{Select Role}

    %% TEACHER WORKFLOW (Setup & Management)
    ROLE -->|Teacher| F0[Teacher Dashboard]
    F0 --> F_CREATE[Create Course & Enrol Students]
    F_CREATE --> F_AGENT[Configure Co-Learning Agent]

    subgraph SETUP [1. Agent Configuration Phase]
        F_AGENT --> F_SETUP1[Upload Syllabus & Content Knowledge Base]
        F_SETUP1 --> F_SETUP2[Define Grading Rubrics & Mastery Metrics]
        F_SETUP2 --> F_SETUP3[Set Guardrails & Pedagogical Persona]
    end

    SETUP --> F_ASSIGN[Publish Assignment / Challenge]
    F_ASSIGN --> F1[View Class Dashboard]

    %% STUDENT WORKFLOW
    ROLE -->|Student| S0[Browse Enrolled Courses]
    S0 --> S1[See Student Dashboard<br/>Subjects, Due Dates & Progress]
    S1 --> S2[Select Assignment & Workspace]

    subgraph SUBMIT [2. Interactive Submission Phase]
        S2 --> S_DRAFT[Work with AI Sandbox/Draft Assistant]
        S_DRAFT --> S_UPLOAD[Upload Final File / Submit Solution]
    end

    S_UPLOAD --> CHECK{Submitted on time?}
    CHECK -->|Yes| TAG1[Status: Submitted]
    CHECK -->|No, past due| TAG2[Status: Late]
    CHECK -->|Not uploaded & past due| TAG3[Status: Missing]

    TAG1 & TAG2 --> AI_ENGINE
    TAG3 --> S_WAIT[Alert: Missing Submission<br/>Submit to unlock feedback]

    %% AI AGENT PIPELINE
    subgraph AI_ENGINE [3. AI Agent Execution Pipeline]
        direction TB
        A0[Parse Submission & Retrieve Context from Knowledge Base]
        A0 --> A1[1. Logic & Accuracy Evaluation]
        A0 --> A2[2. Structural & Presentation Clarity Check]
        A0 --> A3[3. Gap Analysis: Uncovered Concepts]
        A1 & A2 & A3 --> A4[Generate Mastery Vector 0-100%<br/>+ Sanitize Feedback Tone]
    end

    %% STUDENT FEEDBACK LOOP
    AI_ENGINE --> S3[View Detailed Results<br/>Mastery Score + Breakdown]
    S3 --> S4[Review Actionable Feedback & Next Steps]
    S4 --> S5{Mastery Target Met?}
    S5 -->|No| S6[Access Recommended Remedial Content]
    S6 --> S2
    S5 -->|Yes| S_END([End: Target Mastery Achieved])

    %% TEACHER INSIGHTS & INTERVENTION
    F1 --> F2[View Live Completion Rates<br/>Submitted / Late / Missing]
    F2 --> F3[Review Automated Risk Alerts]
    F3 --> F4[Analyze Topic-Wise Class Mastery Heatmap]
    F4 --> F5[Take Direct Action]

    subgraph ACTION [4. Educator Intervention Phase]
        F5 --> ACT1[Adjust Agent Guidance/Prompts]
        F5 --> ACT2[Schedule Target Remediation Class]
        F5 --> ACT3[Override/Refine AI Feedback]
    end

    ACTION --> F6[Dispatch Interventions & Updates]
    F6 --> S3
    F6 --> F_END([End: Continuous Analytics & Support])

    S_END <--> F_END
