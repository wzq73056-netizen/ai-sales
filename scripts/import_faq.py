import json
import pandas as pd

SRC = r"C:\Users\11\Desktop\FAQ卡片知识库_代理记账与SaaS订阅制_200张.xlsx"
OUT_SQL = r"E:\ai\docs\sql\import_faq.sql"

df = pd.read_excel(SRC, sheet_name=0).fillna("")

KB_ID = 2001
KB_NAME = "销售话术FAQ（200张）"


def esc(v: str) -> str:
    return str(v).replace("\\", "\\\\").replace("'", "''").replace("\n", "\\n").replace("\r", "")


lines = ["SET NAMES utf8mb4;"]
lines.append(f"DELETE FROM ai_knowledge_fragment WHERE knowledge_id = {KB_ID};")
lines.append(f"DELETE FROM ai_knowledge WHERE id = {KB_ID};")
lines.append(
    "INSERT INTO ai_knowledge (id, user_id, name, description, share, status, request_count, created_at, updated_at, legacy_table) "
    f"VALUES ({KB_ID}, 1, '{esc(KB_NAME)}', '代理记账与SaaS订阅制销售 FAQ，200 张卡片，来自 Excel 导入', 0, 1, 0, NOW(3), NOW(3), 'excel');"
)

values = []
for idx, row in df.iterrows():
    card_id = str(row["卡片ID"])
    industry = str(row["行业"])
    stage = str(row["销售阶段"])
    question = str(row["客户原话（标准问法）"])
    synonyms = str(row["同义问法"])
    answer = str(row["标准应答话术"])
    logic = str(row["应答逻辑拆解"])
    tags = str(row["分类标签"])
    conf = float(row["置信度"]) if str(row["置信度"]) else 0

    content = (
        f"【{industry}·{stage}】\n"
        f"客户问：{question}\n"
        f"同义问法：{synonyms}\n"
        f"标准应答：{answer}\n"
        f"应答逻辑：{logic}\n"
        f"标签：{tags}"
    )

    values.append(
        f"({KB_ID}, {idx}, '{esc(card_id)}', '{esc(content)}', NOW(3), NOW(3))"
    )

lines.append(
    "INSERT INTO ai_knowledge_fragment (knowledge_id, idx, doc_id, content, created_at, updated_at) VALUES\n"
    + ",\n".join(values)
    + ";"
)

with open(OUT_SQL, "w", encoding="utf-8") as f:
    f.write("\n".join(lines))

print("cards:", len(df), "-> sql written")
print("industries:", df["行业"].value_counts().to_dict())
print("stages:", df["销售阶段"].value_counts().to_dict())
