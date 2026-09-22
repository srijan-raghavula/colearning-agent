flowchart TD
    START([Start: Access Platform]) --> AUTH{Authenticated?}
    AUTH -->|No| LOGIN[Sign Up / Login] --> ROLE
    AUTH -->|Yes| ROLE{Select Role}

    %% ----------------------------------------------------
    %% TEACHER FLOW
    %% ----------------------------------------------------
    ROLE -->|Teacher| F_DASH[Teacher Main Dashboard]

    F_DASH --> F_MENU{Select Action}

    %% Setup Branch
    F_MENU -->|Create / Manage Course| F_COURSE[Course Setup]
    F_COURSE --> F_AGENT_CONFIG[Configure Agent: Knowledge Base, Rubrics & Persona]
    F_AGENT_CONFIG --> F_PUB[Publish Assignment / Challenge]
    F_PUB --> F_DASH

    %% Monitoring & Intervention Branch (Hub & Spoke Model)
    F_MENU -->|View Class Analytics| F_ANALYTICS[Class Overview Hub]

    F_ANALYTICS --> F_VIEW{Choose Analytics View}
    F_VIEW -->|Submissions| F_SUB[Track Status: Submitted / Late / Missing]
    F_VIEW -->|At-Risk Alerts| F_RISK[View At-Risk Student Alerts]
    F_VIEW -->|Mastery Heatmap| F_MAP[Analyze Topic Mastery Heatmap]

    F_SUB & F_RISK & F_MAP --> F_ACTION{Take Action?}
    F_ACTION -->|Adjust Agent| ACT1[Update Prompts / Persona]
    F_ACTION -->|Direct Support| ACT2[Override AI Grade or Send Direct Feedback]
    F_ACTION -->|Class Revision| ACT3[Publish Remedial Material / Schedule Review]
    F_ACTION -->|No Action| F_DASH

    ACT1 & ACT2 & ACT3 --> F_DASH

    %% ----------------------------------------------------
    %% STUDENT FLOW
    %% ----------------------------------------------------
    ROLE -->|Student| S_DASH[Student Main Dashboard]

    S_DASH --> S_MENU{Select Action}

    %% Course Discovery / Enrollment Branch
    S_MENU -->|Browse / Enroll| S_CATALOG[Course Catalog]
    S_CATALOG --> S_JOIN[Enroll with Code or Request Access]
    S_JOIN --> S_DASH

    %% Assignment Working Branch
    S_MENU -->|My Enrolled Courses| S_WORKSPACE[Assignment Workspace]

    S_WORKSPACE --> S_DRAFT[Work with AI Sandbox / Draft Assistant]
    S_DRAFT --> S_SUBMIT[Submit Final Solution]

    S_SUBMIT --> CHECK{Submitted on Time?}
    CHECK -->|Yes| T_YES[Status: Submitted]
    CHECK -->|No, Past Due| T_NO[Status: Late]

    T_YES & T_NO --> AI_ENGINE

    %% ----------------------------------------------------
    %% AI BACKEND ENGINE
    %% ----------------------------------------------------
    subgraph AI_ENGINE [AI Execution Pipeline - Background Process]
        direction TB
        P1[Parse Submission & Match with Knowledge Base]
        P1 --> P2[Evaluate Logic, Clarity & Concept Gaps]
        P2 --> P3[Generate Mastery Scores & Sanitize Feedback]
    end

    %% Feedback & Remediation
    AI_ENGINE --> S_RESULTS[View Feedback & Topic Breakdown]
    S_RESULTS --> S_CHECK{Mastery Achieved?}
    S_CHECK -->|Yes| S_DONE([Complete & Track Growth])
    S_CHECK -->|No| S_REMEDIAL[Review Recommended Remedial Content]
    S_REMEDIAL --> S_WORKSPACE
