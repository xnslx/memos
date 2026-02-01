import { pipeline, type FeatureExtractionPipeline } from "@xenova/transformers";

// =============================================================================
// Local Embedding with BAAI/bge-small-en-v1.5
// =============================================================================

// Model configuration
const MODEL_NAME = "Xenova/bge-small-en-v1.5";
const EMBEDDING_DIMENSIONS = 384;

// Singleton pipeline instance
let embeddingPipeline: FeatureExtractionPipeline | null = null;
let pipelinePromise: Promise<FeatureExtractionPipeline> | null = null;

// Progress callback type
type ProgressCallback = (progress: { status: string; progress?: number }) => void;

// Initialize the embedding pipeline (lazy loading)
async function getEmbeddingPipeline(onProgress?: ProgressCallback): Promise<FeatureExtractionPipeline> {
  if (embeddingPipeline) {
    return embeddingPipeline;
  }

  if (pipelinePromise) {
    return pipelinePromise;
  }

  console.log(`Loading embedding model: ${MODEL_NAME}...`);

  pipelinePromise = pipeline("feature-extraction", MODEL_NAME, {
    progress_callback: onProgress
      ? (progress: { status: string; progress?: number }) => {
          onProgress(progress);
        }
      : undefined,
  });

  embeddingPipeline = await pipelinePromise;
  console.log("Embedding model loaded successfully");

  return embeddingPipeline;
}

// Generate embedding for a single text
export async function generateLocalEmbedding(
  text: string,
  onProgress?: ProgressCallback,
): Promise<number[]> {
  const extractor = await getEmbeddingPipeline(onProgress);

  // bge models recommend adding "query: " or "passage: " prefix for better results
  // For semantic search/clustering, use no prefix or "passage: " for documents
  const output = await extractor(text, {
    pooling: "mean",
    normalize: true,
  });

  // Convert to regular array
  return Array.from(output.data as Float32Array);
}

// Generate embeddings for multiple texts (batched)
export async function generateLocalEmbeddings(
  texts: string[],
  onProgress?: ProgressCallback,
): Promise<number[][]> {
  if (texts.length === 0) return [];

  const extractor = await getEmbeddingPipeline(onProgress);

  const results: number[][] = [];

  // Process in batches to avoid memory issues
  const batchSize = 8;
  for (let i = 0; i < texts.length; i += batchSize) {
    const batch = texts.slice(i, i + batchSize);

    for (const text of batch) {
      const output = await extractor(text, {
        pooling: "mean",
        normalize: true,
      });
      results.push(Array.from(output.data as Float32Array));
    }
  }

  return results;
}

// Check if model is loaded
export function isModelLoaded(): boolean {
  return embeddingPipeline !== null;
}

// Get embedding dimensions
export function getEmbeddingDimensions(): number {
  return EMBEDDING_DIMENSIONS;
}

// Preload the model (call this early for better UX)
export async function preloadEmbeddingModel(onProgress?: ProgressCallback): Promise<void> {
  await getEmbeddingPipeline(onProgress);
}

// Calculate cosine similarity between two embeddings
export function cosineSimilarity(a: number[], b: number[]): number {
  if (!a || !b || a.length !== b.length) return 0;

  let dotProduct = 0;
  let normA = 0;
  let normB = 0;

  for (let i = 0; i < a.length; i++) {
    dotProduct += a[i] * b[i];
    normA += a[i] * a[i];
    normB += b[i] * b[i];
  }

  const denominator = Math.sqrt(normA) * Math.sqrt(normB);
  return denominator === 0 ? 0 : dotProduct / denominator;
}
