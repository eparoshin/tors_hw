#include "options.h"

#include <string_view>
#include <string>
#include <cassert>

namespace NServer {
    TConfig TConfig::Parse(int argc, char** argv) {
        using namespace std::literals;
        TConfig config;

        for (int i = 1; i < argc; ++i) {
            if (std::string_view(argv[i]) == "-p"sv) {
                config.ListenPort = std::stoi(argv[i + 1]);
            }
        }

        return config;
    }
}
