#include "server.h"

namespace NServer {
    TServer::TServer(uint16_t port)
    : port_(port) {
    }

    void TServer::Start(std::function<void (uint16_t)> starting_func) {
        worker_ = std::thread(std::move(starting_func), port_);
    }

    void TServer::Join() {
        worker_.join();
    }

}
