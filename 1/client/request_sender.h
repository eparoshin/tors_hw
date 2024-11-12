#pragma once

#include <netinet/in.h>

#include <vector>
#include <span>

namespace NClient {
    class TRequestSender {
    public:
        explicit TRequestSender(std::span<const sockaddr_in> endpoints);

        std::vector<std::vector<char>> SendRequests(std::span<const std::span<const char>> requests) &&;

        struct TEndpoint {
            sockaddr_in Addr;
            bool Alive = true;
        };

    private:
        TEndpoint& NextEndpoint();

        size_t curr_idx_ = 0;
        std::vector<TEndpoint> endpoints_;
    };
}
