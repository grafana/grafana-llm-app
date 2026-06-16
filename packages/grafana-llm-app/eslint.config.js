import { defineConfig } from "eslint/config";
import rootConfig from "../../eslint.config.js";

export default defineConfig([
    // Extend root configuration
    ...rootConfig,
    
    // Workspace-specific overrides for grafana-llm-app
    {
        files: ["**/*.js", "**/*.jsx", "**/*.ts", "**/*.tsx"],
        rules: {
            "react/prop-types": "off",
        },
    },
]);

