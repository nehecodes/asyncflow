export default [
  {
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "module",
      globals: {
        // browser globals
        window: "readonly",
        document: "readonly",
        console: "readonly"
      }
    },
    rules: {}
  }
];