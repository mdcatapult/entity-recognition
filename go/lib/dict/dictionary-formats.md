# NER dictionary formats

### Glossary
* **Identifiers** (map string to string): stable identifiers/accession codes that we can use to provide links to external resources.
* **Synonyms** (string array): All the terms that we want to recognize and resolve to the given identifiers.
* **metadata** (map string to any): Any other metadata that we think might be useful. *For now, metadata is string to string only!*


### API Response
**POST /html/entities** 
```json
[
    {
        "entity": "P0C090",
        "dictionary": "gene_protein",
        "identifiers": {
            "swissprot_accession": "P0C090",
            "swissprot_id": "1",
            "gene_name": "dave"
        },
        "metadata": {
            "length": 824084084084,
        }
    }
]
```

### Input files

* Native format (json lines file):
e.g.`gene_protein.jsonl`
```json
{"synonyms": ["P0C090"], "identifiers": {"swissprot_accession": "P0C090","swissprot_id": "1","gene_name": "dave"}, "metadata": {"length": 824084084084}}
{...}
```
* [Leadmine format](../../cmd/dictionary-importer/dictionaries/leadmine.tsv).
* [Pubchem format](../../cmd/dictionary-importer/dictionaries/pubchem.tsv).
* [Swissprot format](../../cmd/dictionary-importer/dictionaries/swissprot.jsonl).