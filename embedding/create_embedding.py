import argparse
import os
import sys
import numpy as np
import torch
import torch.nn.functional as F
from transformers import AutoTokenizer, AutoModel
from tqdm import tqdm

def mean_pooling(model_output, attention_mask):
    token_embeddings = model_output[0]
    input_mask_expanded = attention_mask.unsqueeze(-1).expand(token_embeddings.size()).float()
    return torch.sum(token_embeddings * input_mask_expanded, 1) / torch.clamp(input_mask_expanded.sum(1), min=1e-9)

def get_device():
    if torch.backends.mps.is_available():
        return torch.device("mps")
    elif torch.cuda.is_available():
        return torch.device("cuda")
    else:
        return torch.device("cpu")

def load_data(filepath, limit=None):
    """
    Generator that yields (doc_id, text) tuples from the TSV file.
    """
    print(f"Loading data from {filepath}...")
    with open(filepath, 'r', encoding='utf-8') as f:
        count = 0
        for line in f:
            if limit and count >= limit:
                break
            
            parts = line.strip().split('\t')
            if len(parts) >= 2:
                # Assuming format: doc_id \t url \t title \t body 
                # OR doc_id \t content
                # We need to construct the text to embed.
                # Based on typical MS MARCO: ID, URL, Title, Body
                
                doc_id_str = parts[0]
                try:
                    doc_id = int(doc_id_str)
                except ValueError:
                    # Skip header or malformed lines if any
                    continue
                
                if len(parts) >= 4:
                    title = parts[2]
                    # Join all remaining parts as body, similar to the Go loader
                    # checking for any accidental tabs in the body
                    body = " ".join(parts[3:])
                    text = f"search_document: {title} {body}"
                else:
                    # Fallback
                    text = f"search_document: {' '.join(parts[1:])}"
                
                yield doc_id, text
                count += 1

def main():
    parser = argparse.ArgumentParser(description="Generate embeddings for MS MARCO dataset")
    parser.add_argument("--data", type=str, required=True, help="Path to input TSV file")
    parser.add_argument("--output-dir", type=str, required=True, help="Directory to save embeddings")
    parser.add_argument("--batch-size", type=int, default=64, help="Batch size for inference")
    parser.add_argument("--limit", type=int, default=None, help="Limit number of documents to process")
    parser.add_argument("--max-length", type=int, default=2048, help="Max sequence length (default: 2048 to save memory)")
    
    args = parser.parse_args()
    
    os.makedirs(args.output_dir, exist_ok=True)
    
    device = get_device()
    print(f"Using device: {device}")
    
    print("Loading model nomic-ai/nomic-embed-text-v1.5...")
    tokenizer = AutoTokenizer.from_pretrained("nomic-ai/nomic-embed-text-v1.5", trust_remote_code=True)
    # Load model in float16 to save memory and avoid MPS buffer limits
    model = AutoModel.from_pretrained("nomic-ai/nomic-embed-text-v1.5", trust_remote_code=True, dtype=torch.float16)
    model.to(device)
    model.eval()
    
    doc_ids = []
    embeddings = []
    
    batch_texts = []
    batch_ids = []
    
    # Estimate total for tqdm if possible (or just use simple counter)
    # Total MS MARCO docs is ~3.2M, or use limit if set
    total_docs = 3213835
    if args.limit:
        total_docs = args.limit
    
    data_gen = load_data(args.data, args.limit)
    
    for doc_id, text in tqdm(data_gen, total=total_docs, desc="Processing documents"):
        batch_ids.append(doc_id)
        batch_texts.append(text)
        
        if len(batch_texts) >= args.batch_size:
            # Process batch
            encoded_input = tokenizer(
                batch_texts, 
                padding=True, 
                truncation=True, 
                max_length=args.max_length, 
                return_tensors='pt'
            )
            encoded_input = {k: v.to(device) for k, v in encoded_input.items()}
            
            with torch.no_grad():
                model_output = model(**encoded_input)
                
            emb = mean_pooling(model_output, encoded_input['attention_mask'])
            emb = F.layer_norm(emb, normalized_shape=(emb.shape[1],))
            emb = F.normalize(emb, p=2, dim=1)
            
            embeddings.append(emb.cpu().numpy())
            doc_ids.extend(batch_ids)
            
            batch_texts = []
            batch_ids = []
            
    # Process remaining
    if batch_texts:
        encoded_input = tokenizer(
            batch_texts, 
            padding=True, 
            truncation=True, 
            max_length=args.max_length, 
            return_tensors='pt'
        )
        encoded_input = {k: v.to(device) for k, v in encoded_input.items()}
        
        with torch.no_grad():
            model_output = model(**encoded_input)
            
        emb = mean_pooling(model_output, encoded_input['attention_mask'])
        emb = F.layer_norm(emb, normalized_shape=(emb.shape[1],))
        emb = F.normalize(emb, p=2, dim=1)
        
        embeddings.append(emb.cpu().numpy())
        doc_ids.extend(batch_ids)
        
    print("Concatenating embeddings...")
    if embeddings:
        all_embeddings = np.concatenate(embeddings, axis=0)
        all_doc_ids = np.array(doc_ids, dtype=np.uint32) # Use uint32 (max 4M docs)
        
        print(f"Saving to {args.output_dir}...")
        print(f"Embeddings shape: {all_embeddings.shape}")
        print(f"Doc IDs shape: {all_doc_ids.shape}")
        
        np.save(os.path.join(args.output_dir, "embeddings.npy"), all_embeddings)
        np.save(os.path.join(args.output_dir, "doc_ids.npy"), all_doc_ids)
        print("Done.")
    else:
        print("No embeddings generated.")

if __name__ == "__main__":
    main()
