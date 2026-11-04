# ShardStore (SOSP'21) mapped to dst

Bornholt et al., *Finding and Preventing Bugs in Systems through Deterministic Simulation Testing*, SOSP 2021, centers on ShardStore. This note maps the paper's ideas onto this module. It is not a full proof of any product.

## Reference model

The paper keeps a sequential reference model and compares it to the system under test. Package `model` runs model and subject in lockstep with `Step.Check`. Observations must match; an optional invariant runs after each op. Callers own the model state and the op alphabet.

## Property-based testing lockstep

ShardStore generates operation sequences and checks each step. `model` stays free of a PBT engine so `go.mod` stays small. Pass ops from rapid (or any generator) at the call site. Put simplest ops first in the alphabet so shrinkers prefer short failing prefixes.

## Shrink

When a long sequence fails, shrink toward a shorter one that still fails. Ordered alphabets and pure ops make shrink cheap. dst does not ship a shrinker; the host test or rapid does.

## Crash property

After a crash and recover, answered writes must still be present, and a clean restart must match the model again. `model.CheckPersistence` and `model.CheckForwardProgress` take callbacks so product types stay out of this module. Issue-time crash faults belong in `fault` / `sim` Crash+Restart.

## What this is not

dst does not prove linearizability, invent a product Machine type, or replace a full network-partition simulator. It hosts deterministic runs, fingerprints (`trace`, `golden`), and lockstep checks so a product can encode its own ShardStore-style campaign.
