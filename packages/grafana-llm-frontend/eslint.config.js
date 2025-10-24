import { defineConfig } from "eslint/config";
import rootConfig from "../../eslint.config.js";

export default defineConfig([
    // Extend root configuration
    ...rootConfig,
    
    // grafana-llm-frontend uses base config with no additional overrides
]);

