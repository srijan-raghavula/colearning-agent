flowchart TD
    START([Open CoLearn]) --> ROLE{Choose a workspace}

    %% ----------------------------------------------------
    %% TEACHER FLOW
    %% ----------------------------------------------------
    ROLE -->|Teacher| T_DASH[Teacher class dashboard]
    T_DASH --> T_VIEW{Choose a teacher action}
    T_VIEW -->|Shape learning context| T_CONTENT[Content studio]
    T_CONTENT --> T_AUTHOR[Author concept material]
    T_AUTHOR --> T_PUBLISH[Publish versioned material]
    T_PUBLISH --> T_DASH

    T_VIEW -->|Configure tutor| T_POLICY[Choose allowed tutor modes]
    T_POLICY --> T_SOURCE[Set teacher + external source policy]
    T_SOURCE --> T_DASH

    T_VIEW -->|See learning signals| T_SIGNALS[Concept confidence and evidence]
    T_SIGNALS --> T_SUPPORT{Student needs support?}
    T_SUPPORT -->|Yes| T_GUIDE[Add guidance or adjust context]
    T_SUPPORT -->|No| T_DASH
    T_GUIDE --> T_DASH

    %% ----------------------------------------------------
    %% STUDENT FLOW
    %% ----------------------------------------------------
    ROLE -->|Student| S_DASH[Student learning dashboard]
    S_DASH --> S_PATH[Choose a concept or resume active session]
    S_PATH --> S_SESSION[Start or resume learning session]
    S_SESSION --> S_SPACE[Open tutor learning space]

    S_SPACE --> S_TURN[Ask, explain, request a hint, or practice]
    S_TURN --> S_MODE{Choose tutor mode}
    S_MODE -->|Guided| S_GUIDED[Guided explanation and next example]
    S_MODE -->|Socratic| S_SOCRATIC[Question and learner explanation]
    S_MODE -->|Direct| S_DIRECT[Direct explanation]
    S_MODE -->|Practice| S_PRACTICE[Practice prompt]
    S_MODE -->|Hint| S_HINT[Progressive hint]
    S_MODE -->|Diagnostic| S_DIAGNOSTIC[Understand current starting point]

    S_GUIDED --> S_SAVE[Persist turns and learning evidence]
    S_SOCRATIC --> S_SAVE
    S_DIRECT --> S_SAVE
    S_PRACTICE --> S_SAVE
    S_HINT --> S_SAVE
    S_DIAGNOSTIC --> S_SAVE

    S_SAVE --> S_PROGRESS[Update concept confidence and next action]
    S_PROGRESS --> S_CONTINUE{Continue learning?}
    S_CONTINUE -->|Yes| S_SPACE
    S_CONTINUE -->|Pause and return later| S_DASH

    %% ----------------------------------------------------
    %% OPTIONAL FORMATIVE CHECK
    %% ----------------------------------------------------
    S_SPACE -. Student or teacher requests a check .-> CHECK[Optional understanding check]
    CHECK --> S_EVIDENCE[Store formative evidence]
    S_EVIDENCE --> S_PROGRESS
