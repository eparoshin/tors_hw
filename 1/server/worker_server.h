#pragma once

#include "server.h"

namespace NServer {
class TWorkerServer : public TServer {
    public:
        explicit TWorkerServer(uint16_t port);

        void Start();
    };
}
