import nextCoreWebVitals from "eslint-config-next/core-web-vitals";

const eslintConfig = [
  {
    ignores: ["node_modules/", ".next/", "out/", "coverage/"],
  },
  ...nextCoreWebVitals,
  {
    rules: {
      // TODO: react-hooks v6 (React Compiler) rules; downgrade to warn until
      // the data-fetching hooks (useCoinHistory, useMarketData) and the SSE
      // reconnect logic (useMarketStream) are refactored to comply.
      "react-hooks/set-state-in-effect": "warn",
      "react-hooks/immutability": "warn",
    },
  },
];

export default eslintConfig;
