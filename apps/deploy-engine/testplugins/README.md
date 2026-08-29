# Test plugins

Plugins used to exercise the deploy engine locally.
They do not talk to any upstream system, deploying resources with them reports
success without doing anything.

## Example provider

`exampleprovider` provides the `example` namespace:

| Item                     | Kind                 | Options / fields                                                                     |
| ------------------------ | -------------------- | ------------------------------------------------------------------------------------ |
| `example/region`         | custom variable type | `us-east-1`, `us-east-2`, `us-west-1`, `us-west-2`, `eu-west-1`, `eu-west-2`, `eu-central-1` |
| `example/instanceSize`   | custom variable type | `small`, `medium`, `large`, `xlarge`                                                   |
| `example/environment`    | custom variable type | `development`, `staging`, `production`                                                 |
| `example/service`        | resource             | `serviceName` (required), `region`, `id` (computed)                                    |

### Building

```bash
bash scripts/build-test-plugins.sh
```

This installs the plugin at:

```
.bluelink/deploy-engine/plugins/bin/providers/newstack-cloud/example/0.1.0/plugin
```

which is the plugin directory used by `scripts/run-local.sh --host`.
Restart the deploy engine to pick it up.

To build real plugins from their own source repositories into the same plugin
directory, use `scripts/build-local-plugins.sh`, see
[Building plugins for local testing](../docs/CONTRIBUTING.md#building-plugins-for-local-testing).

The requests below assume the API key auth used by `.env.example.host`
(`BLUELINK_DEPLOY_ENGINE_AUTH_BLUELINK_API_KEYS=test-api-key`), adjust the
`Bluelink-Api-Key` header to match your local configuration.

### Checking suggestions for an unknown custom variable type

Validate `exampleprovider/__testdata/unknown-variable-type.blueprint.yaml`,
which declares a variable of type `example/regionn`.

```bash
curl -X POST http://localhost:8325/v1/validations \
  -H 'Content-Type: application/json' \
  -H 'Bluelink-Api-Key: test-api-key' \
  -d '{
    "directory": "'"$PWD"'/testplugins/exampleprovider/__testdata",
    "blueprintFile": "unknown-variable-type.blueprint.yaml"
  }'
```

The diagnostic for the variable carries `suggestions` of `["example/region"]`
and `availableValues` of the three custom variable types in the
`context.metadata` of the streamed event.

### Checking suggestions for a value that is not one of the options

The options for a custom variable type are only checked against a value when
the loader validates runtime values, which the validation endpoint does when
`checkBlueprintVars=true` is set. The value comes from `config.blueprintVariables`.

Validate `exampleprovider/__testdata/value-not-in-options.blueprint.yaml` with a
region that is close to one of the options but is not one of them:

```bash
curl -X POST 'http://localhost:8325/v1/validations?checkBlueprintVars=true' \
  -H 'Content-Type: application/json' \
  -H 'Bluelink-Api-Key: test-api-key' \
  -d '{
    "directory": "'"$PWD"'/testplugins/exampleprovider/__testdata",
    "blueprintFile": "value-not-in-options.blueprint.yaml",
    "config": {
      "blueprintVariables": {
        "region": "eu-central-4"
      }
    }
  }'
```

The diagnostic carries `suggestions` of `["eu-central-1"]` and
`availableValues` of the seven regions. `eu-west-1` and `eu-west-2` are within
the default similarity threshold for a value of this length but are excluded by
the tighter threshold used for suggestions, so a single suggestion is expected
here.

To check that unrelated values produce no suggestions at all, use a region such
as `moon-base-1`, which is far enough from every option that only
`availableValues` is reported.
