#/bin/bash -x

## TEST_CONFIGS are the butler toml's that are to be retrieved
TEST_CONFIGS=(http://localhost/toml/butler1.toml https://localhost/toml/butler1.toml https://localhost/toml/butler2.toml https://127.0.0.1/toml/butler2.toml https://localhost/toml/butler3.toml https://localhost/
toml/butler4.toml https://localhost/toml/butler5.toml https://localhost/toml/butler6.toml https://localhost/toml/butler7.toml https://localhost/toml/butler8.toml https://localhost/toml/butler9.toml https://local
host/toml/butler10.toml file:///www/toml/butler11.toml)

## TEST_RESPONSES are the unix return code that should be recieved by butler for each config. the seqence of the number in the array should match up w/ the expected respose of the config in TEST_CONFIGS
TEST_RESPONSES=(1 1 1 1 1 1 1 1 1 0 0 0 1 1 0)
ITER=0

echo "[testing main configs]"
for config in "${TEST_CONFIGS[@]}"
do
    cmd="/butler -config.path $config -config.retrieve-interval 10 -log.level debug"
    echo "  [testing $cmd]"
    $cmd -test
    res=$?
    if [ $res -ne ${TEST_RESPONSES[$((ITER))]} ]; then
        echo "unexpected result for $cmd. res=$res expecting res to be ${TEST_RESPONSES[$((ITER))]} at array place $ITER (starting at 0)"
        exit 1
    fi
    ITER=$((ITER+1))
    echo "  [done testing $cmd]"
    echo
done
echo "[done testing main configs]"
echo

