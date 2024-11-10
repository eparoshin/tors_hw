#pragma once

#include "server.h"

namespace NServer {
    class TDiscoveryServer : public TServer {
    public:
        explicit TDiscoveryServer(uint16_t port);

        void Start();
    };
}
