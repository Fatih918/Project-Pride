module pride-plugin-module-sdkadapter

go 1.26.6

require (
	pride-plugin-module v0.0.0
	pride-plugin-sdk v0.0.0
)

replace pride-plugin-module => ..

replace pride-plugin-sdk => ../../../sdks/plugin
