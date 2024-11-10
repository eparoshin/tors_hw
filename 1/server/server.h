#pragma once

#include <thread>
#include <cstdint>
#include <functional>

namespace NServer {
    class TServer {
    public:
        explicit TServer(uint16_t port);

        void Join();

    protected:
        void Start(std::function<void (uint16_t)> starting_func);
    private:
        const uint16_t port_;
        std::thread worker_;
    };
}
