**Diagrams**

- [Preparing a Virtual Environment](#preparing-a-virtual-environment)
- [Generate Diagrams](#generate-diagrams)

---

# Preparing a Virtual Environment

1. Create a virtual environment:
```bash
python -m venv ./venv/
```
2. Activate the virtual environment:
```bash
source ./venv/bin/activate
```
3. Install dependencies:
```bash
pip install -Ur ./docs/diagrams/requirements.txt
```

# Generate Diagrams

```bash
python ./docs/diagrams/main.py
```
