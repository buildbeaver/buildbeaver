module.exports = {
    presets: [
        [
            '@babel/preset-env',
            {
                useBuiltIns: "usage",
                corejs: 3,
                debug: false,
            },
        ],
        [
            '@babel/preset-typescript',
            {},
        ],
        [
            '@babel/preset-react',
            {
                runtime: 'automatic',
            },
        ],
    ]
};
