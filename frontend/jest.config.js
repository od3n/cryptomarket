const nextJest = require("next/jest");

const createJestConfig = nextJest({
  dir: "./",
});

/** @type {import('jest').Config} */
const customJestConfig = {
  testEnvironment: "jsdom",
  setupFilesAfterEnv: ["<rootDir>/jest.setup.ts"],
  moduleNameMapper: {
    "^@/(.*)$": "<rootDir>/$1",
  },
  testPathIgnorePatterns: ["<rootDir>/node_modules/", "<rootDir>/.next/", "<rootDir>/e2e/"],
  modulePathIgnorePatterns: ["<rootDir>/.next/"],
  collectCoverageFrom: [
    "**/*.{ts,tsx}",
    "!**/*.d.ts",
    "!**/node_modules/**",
    "!**/.next/**",
  ],
  // Ratchet floor: set to the current coverage baseline to prevent
  // regressions. Raise incrementally toward the 75%/70% target as tests
  // are added (components and hooks are the least-covered areas).
  coverageThreshold: {
    global: {
      branches: 25,
      functions: 22,
      lines: 20,
      statements: 20,
    },
  },
};

module.exports = createJestConfig(customJestConfig);
