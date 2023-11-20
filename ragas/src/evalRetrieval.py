from ragas import evaluate
from datasets import load_dataset
import os

os.environ["OPENAI_API_KEY"] = "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"


def evalRetrieval():
    fiqa_eval = load_dataset("explodinggradients/fiqa", "ragas_eval")["baseline"]

    results = evaluate(fiqa_eval)
    return results
    # {'ragas_score': 0.860, 'context_precision': 0.817,
    # 'faithfulness': 0.892, 'answer_relevancy': 0.874}
