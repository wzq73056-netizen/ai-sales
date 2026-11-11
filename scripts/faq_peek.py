import json
import pandas as pd

path = r"C:\Users\11\Desktop\FAQ卡片知识库_代理记账与SaaS订阅制_200张.xlsx"
x = pd.ExcelFile(path)
df = x.parse(x.sheet_names[0])

out = {
    "sheets": x.sheet_names,
    "columns": [str(c) for c in df.columns],
    "rows": df.head(5).fillna("").to_dict(orient="records"),
    "total": len(df),
    "value_counts": {
        str(k): {str(kk): int(vv) for kk, vv in v.items()}
        for k, v in [("行业", df.iloc[:, 1].value_counts()), ("客户异议类型", df.iloc[:, 3].value_counts())]
    },
}

with open(r"E:\ai\docs\faq_peek.json", "w", encoding="utf-8") as f:
    json.dump(out, f, ensure_ascii=False, indent=2, default=str)

print("written", len(df))
