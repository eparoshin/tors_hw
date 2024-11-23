#pragma once

#include "rcu.h"

#include <netinet/in.h>

#include <vector>
#include <span>

namespace NClient {
    class TRequestSender {
    public:
        explicit TRequestSender(const NUtil::TRcu<std::vector<sockaddr_in>>& addrsRef, size_t maxUnupdated);

        std::vector<std::vector<char>> SendRequests(std::span<const std::span<const char>> requests) &&;

        struct TEndpoint {
            sockaddr_in Addr;
            bool Alive = true;
        };

    private:
        std::shared_ptr<TEndpoint> NextEndpoint();

        void UpdateEndpoints();

        const NUtil::TRcu<std::vector<sockaddr_in>>& addrsRef_;
        const size_t maxUnupdated_;
        size_t curr_idx_ = 0;
        size_t numUpdates_ = 0;
        std::vector<std::shared_ptr<TEndpoint>> endpoints_;
    };
}
