#pragma once

#include "rcu.h"

#include <netinet/in.h>

#include <cstdint>
#include <vector>
#include <span>

namespace NClient {
    struct TPoint {
        double x;
        double y;
    };

    class TDiscovery {
    public:
        TDiscovery(const TDiscovery&) = delete;
        TDiscovery& operator=(const TDiscovery&) = delete;

        TDiscovery(in_addr broadcastAddr, uint16_t port);

        void UpdateList(int timeout);

        double Calc(std::span<const TPoint> points) const;

        ~TDiscovery();
    private:
        in_addr broadcastAddr_;
        uint16_t port_;
        int sockFd_;
        NUtil::TRcu<std::vector<sockaddr_in>> addrsRef_;
    };
}
