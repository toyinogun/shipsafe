export const NAV_LINKS = [
  { label: "How it Works", href: "#how-it-works" },
  { label: "Features", href: "#features" },
  { label: "Integrations", href: "#integrations" },
  { label: "Pricing", href: "#pricing" },
] as const;

export const HERO = {
  badge: "v0.3.0 is now live",
  headline: "Ship AI-Generated Code",
  headlineAccent: "with Confidence",
  subheadline:
    "The self-hosted gateway that verifies AI code before it hits production. Detect hallucinations, security vulnerabilities, and logic flaws instantly.",
  primaryCta: "Start Scanning Free",
  secondaryCta: "View Documentation",
} as const;

export const TERMINAL_DEMO = {
  title: "shipsafe-cli \u2014 scan",
  command: "shipsafe scan --target ./src/utils/ai-helper.ts",
  initLines: [
    "Initializing analysis engine...",
    "Loading context from repo...",
  ],
  filename: "ai-helper.ts",
  resultLabel: "Analysis Complete",
  resultTitle: "Code Integrity Verified",
  resultSubtitle: "No hallucinations or critical vulnerabilities found.",
  score: 92,
  status: "PASSED" as const,
} as const;

export const LOGO_CLOUD = {
  heading: "Trusted by engineering teams at",
  logos: [
    { icon: "code", name: "DevCorp" },
    { icon: "cloud_queue", name: "CloudScale" },
    { icon: "data_object", name: "DataFlow" },
    { icon: "security", name: "SecureOps" },
    { icon: "api", name: "APIMesh" },
  ],
} as const;

export const STEPS = [
  {
    icon: "hub",
    title: "1. Connect Repo",
    description:
      "Install the ShipSafe bot on your GitHub or GitLab repositories with one click.",
  },
  {
    icon: "psychology",
    title: "2. AI Analysis",
    description:
      "Our engine scans every PR for AI-generated patterns and potential logic hallucinations.",
  },
  {
    icon: "verified_user",
    title: "3. Safe Merge",
    description:
      "Get a green light trust score. Merge with confidence knowing the code is verified.",
  },
] as const;

export const FEATURES = [
  {
    icon: "security",
    title: "Self-Hosted Security",
    description: "Keep your code private. Run ShipSafe within your own VPC.",
  },
  {
    icon: "bug_report",
    title: "Hallucination Detection",
    description: "Spot non-existent libraries and logic flaws instantly.",
  },
  {
    icon: "sync",
    title: "CI/CD Pipeline Native",
    description:
      "Blocks builds if the trust score falls below your threshold.",
  },
  {
    icon: "rule",
    title: "Custom Rule Engine",
    description:
      'Define what "safe" means for your specific architecture.',
  },
] as const;

export const TRUST_SCORE_CARDS = [
  {
    status: "red" as const,
    icon: "warning",
    pr: "PR #402",
    repo: "payment-service",
    label: "RISK",
    score: 32,
  },
  {
    status: "yellow" as const,
    icon: "error_outline",
    pr: "PR #405",
    repo: "frontend-auth",
    label: "WARN",
    score: 65,
  },
  {
    status: "green" as const,
    icon: "check_circle",
    pr: "PR #409",
    repo: "database-migration",
    label: "Excellent",
    score: 98,
    author: "Copilot",
    metrics: [
      { label: "Logic Consistency", value: 100 },
      { label: "Security Scan", value: 96 },
    ],
  },
] as const;

export const INTEGRATIONS = [
  { icon: "code", name: "VS Code" },
  { icon: "folder_open", name: "GitHub Actions" },
  { icon: "cloud_circle", name: "GitLab CI" },
  { icon: "terminal", name: "CLI" },
] as const;

export const CTA = {
  headline: "Ready to secure your AI workflow?",
  subheadline: "Join 500+ engineering teams shipping better code, faster.",
  primaryCta: "Get Started for Free",
  secondaryCta: "Talk to Sales",
} as const;

export const FOOTER = {
  tagline: "The standard for AI code verification.",
  columns: [
    {
      title: "Product",
      links: [
        { label: "Features", href: "#features" },
        { label: "Integrations", href: "#integrations" },
        { label: "Pricing", href: "#pricing" },
        { label: "Changelog", href: "#" },
      ],
    },
    {
      title: "Resources",
      links: [
        { label: "Documentation", href: "#" },
        { label: "API Reference", href: "#" },
        { label: "Community", href: "#" },
        { label: "Blog", href: "#" },
      ],
    },
    {
      title: "Legal",
      links: [
        { label: "Privacy", href: "#" },
        { label: "Terms", href: "#" },
        { label: "Security", href: "#" },
      ],
    },
  ],
  copyright: "\u00a9 2025 ShipSafe. All rights reserved.",
} as const;
