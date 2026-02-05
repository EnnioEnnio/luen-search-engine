import concurrent.futures
import os

import grpc
import numpy as np
import torch
import torch.nn.functional as F
from transformers import AutoModel, AutoTokenizer

import search_pb2
import search_pb2_grpc

MAX_WORKERS = 10
PORT = 50051


def get_device():
    if torch.backends.mps.is_available():
        return torch.device("mps")
    elif torch.cuda.is_available():
        return torch.device("cuda")
    else:
        return torch.device("cpu")


def mean_pooling(model_output, attention_mask):
    token_embeddings = model_output[0]
    input_mask_expanded = (
        attention_mask.unsqueeze(-1).expand(token_embeddings.size()).float()
    )
    return torch.sum(token_embeddings * input_mask_expanded, 1) / torch.clamp(
        input_mask_expanded.sum(1), min=1e-9
    )


class SemanticSearchService(search_pb2_grpc.SemanticEmbeddingServiceServicer):
    def __init__(self, embedding_dir):
        print(f"Loading resources from {embedding_dir}...")
        try:
            self.embeddings = np.load(os.path.join(embedding_dir, "embeddings.npy"))
            self.doc_ids = np.load(os.path.join(embedding_dir, "doc_ids.npy"))
            print(f"Loaded {self.embeddings.shape[0]} documents.")
        except FileNotFoundError:
            print("WARNING: Embedding files not found. Search will return empty.")
            self.embeddings = None
            self.doc_ids = None

        self.device = get_device()
        print(f"Using device: {self.device}")

        # Load model with same config as creation script
        # Using float32 for inference/server might be safer for compatibility,
        # but float16 is faster if supported. We use float32 for now to avoid
        # potential type mismatches with numpy.
        print("Loading model...")
        self.tokenizer = AutoTokenizer.from_pretrained(
            "nomic-ai/nomic-embed-text-v1.5", trust_remote_code=True
        )
        self.model = AutoModel.from_pretrained(
            "nomic-ai/nomic-embed-text-v1.5", trust_remote_code=True
        )
        self.model.to(self.device)
        self.model.eval()
        print("Service ready.")

    def Search(self, request, context):
        if self.embeddings is None:
            return search_pb2.SearchResponse(results=[])

        query = request.query
        k = request.k
        if k <= 0:
            k = 10

        # Nomic specific prefix
        query_text = f"search_query: {query}"

        # Embed query
        encoded_input = self.tokenizer(
            [query_text], padding=True, truncation=True, return_tensors="pt"
        )
        encoded_input = {k: v.to(self.device) for k, v in encoded_input.items()}

        with torch.no_grad():
            model_output = self.model(**encoded_input)

        emb = mean_pooling(model_output, encoded_input["attention_mask"])
        emb = F.layer_norm(emb, normalized_shape=(emb.shape[1],))

        # Truncate to match stored embedding dimension if necessary (Matryoshka)
        stored_dim = self.embeddings.shape[1]
        if stored_dim < emb.shape[1]:
            emb = emb[:, :stored_dim]

        emb = F.normalize(emb, p=2, dim=1)

        query_vec = emb.cpu().numpy()[0]  # Shape (stored_dim,)

        # Calculate scores (Dot product)
        # matrix: (N, stored_dim), query: (stored_dim,) -> scores: (N,)
        scores = np.dot(self.embeddings, query_vec)

        # Find top k
        # argpartition is O(n) average case
        if k >= len(scores):
            top_indices = np.argsort(scores)[::-1]
        else:
            partitioned_indices = np.argpartition(scores, -k)[-k:]
            # The top k are not sorted, so we sort them explicitly
            sorted_indices_in_top_k = np.argsort(scores[partitioned_indices])[::-1]
            top_indices = partitioned_indices[sorted_indices_in_top_k]

        results = []
        for idx in top_indices:
            results.append(
                search_pb2.SearchResponse.Result(
                    doc_id=self.doc_ids[idx], score=float(scores[idx])
                )
            )

        return search_pb2.SearchResponse(results=results)


def serve():
    server = grpc.server(concurrent.futures.ThreadPoolExecutor(max_workers=MAX_WORKERS))
    search_pb2_grpc.add_SemanticEmbeddingServiceServicer_to_server(
        SemanticSearchService("output"), server
    )
    server.add_insecure_port(f"[::]:{PORT}")
    print(f"Server starting on port {PORT}...")
    server.start()
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
