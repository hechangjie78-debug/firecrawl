import { createOpenAI } from "@ai-sdk/openai";
import { config } from "../config";
import { createOllama } from "ollama-ai-provider";
import { anthropic } from "@ai-sdk/anthropic";
import { groq } from "@ai-sdk/groq";
import { google } from "@ai-sdk/google";
import { createOpenRouter } from "@openrouter/ai-sdk-provider";
import { fireworks } from "@ai-sdk/fireworks";
import { deepinfra } from "@ai-sdk/deepinfra";
import { createVertex } from "@ai-sdk/google-vertex";
import type { LanguageModelV2 } from "@ai-sdk/provider";

const OPENCODE_ZEN = !!(
  config.OPENAI_BASE_URL?.includes("opencode.ai") ||
  config.OPENAI_BASE_URL?.includes("zen")
);

function wrapModel(model: LanguageModelV2): LanguageModelV2 {
  if (!OPENCODE_ZEN) return model;
  return {
    ...model,
    doGenerate: async (options: any) => {
      if (options.responseFormat?.type === "json_schema") {
        return model.doGenerate({
          ...options,
          responseFormat: { type: "json_object" },
        });
      }
      return model.doGenerate(options);
    },
  };
}

type Provider =
  | "openai"
  | "ollama"
  | "anthropic"
  | "groq"
  | "google"
  | "openrouter"
  | "fireworks"
  | "deepinfra"
  | "vertex";
const defaultProvider: Provider = config.OLLAMA_BASE_URL ? "ollama" : "openai";

const providerList: Record<Provider, any> = {
  openai: createOpenAI({
    apiKey: config.OPENAI_API_KEY,
    baseURL: config.OPENAI_BASE_URL,
  }), //OPENAI_API_KEY
  ollama: createOllama({
    baseURL: config.OLLAMA_BASE_URL,
  }),
  anthropic, //ANTHROPIC_API_KEY
  groq, //GROQ_API_KEY
  google, //GOOGLE_GENERATIVE_AI_API_KEY
  openrouter: createOpenRouter({
    apiKey: config.OPENROUTER_API_KEY,
  }),
  fireworks, //FIREWORKS_API_KEY
  deepinfra, //DEEPINFRA_API_KEY
  vertex: createVertex({
    project: "firecrawl",
    //https://github.com/vercel/ai/issues/6644 bug
    baseURL:
      "https://aiplatform.googleapis.com/v1/projects/firecrawl/locations/global/publishers/google",
    location: "global",
    googleAuthOptions: config.VERTEX_CREDENTIALS
      ? {
          credentials: JSON.parse(atob(config.VERTEX_CREDENTIALS)),
        }
      : {
          keyFile: "./gke-key.json",
        },
  }),
};

export function getModel(name: string, provider: Provider = defaultProvider) {
  if (name === "gemini-2.5-pro") {
    name = "gemini-2.5-pro";
  }
  const modelName = config.MODEL_NAME || name;
  // o3-mini returns empty text via the Responses API — force Chat Completions
  if (provider === "openai" && modelName.startsWith("o3-mini")) {
    return wrapModel(providerList.openai.chat(modelName));
  }
  // OpenCode Zen 免费模型仅支持 Chat Completions，不支持 Responses API
  if (OPENCODE_ZEN) {
    return wrapModel(providerList.openai.chat(modelName));
  }
  return providerList[provider](modelName);
}

export function getEmbeddingModel(
  name: string,
  provider: Provider = defaultProvider,
) {
  return config.MODEL_EMBEDDING_NAME
    ? providerList[provider].embedding(config.MODEL_EMBEDDING_NAME)
    : providerList[provider].embedding(name);
}
