# DHT2VEC

A feature addressable distributed hash table using vectors to search for similarity across a peer-to-peer network. Returning the most similar file.
For a more detail read, check out the [blog post](https://systemshift.github.io/FAN.html)


# How to use

Project currently in prototype stage, and what exists is the most basic implementation in a controlled environment.

```
python lookup.py [filepath]
```

The datalookup has `DATA`, which then has many directories as `DATA/something/`

# Requirements

```
Tensorflow
Numpy
Pillow
```

# TODO

There is no network interface right now, and DHT implementations that exist have hashing baked in. So a DHT needs to be implemented from scratch with the new primitives hardcoded in the codebase.