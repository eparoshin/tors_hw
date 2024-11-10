#pragma once

#include <cstdint>

namespace NServer {
    struct TConfig {
        static TConfig Parse(int argc, char** argv);

        uint16_t ListenPort = 12345;
    };
}
