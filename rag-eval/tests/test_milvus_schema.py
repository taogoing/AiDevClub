from rag_eval.milvus_schema import chinese_analyzer_params


def test_milvus_jieba_is_declared_as_the_analyzer_tokenizer():
    assert chinese_analyzer_params() == {"tokenizer": "jieba"}
