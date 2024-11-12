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
        //TODO subnet
        explicit TDiscovery(uint16_t port);

        void UpdateList();

        double Calc(std::span<const TPoint> points) const;
    private:
        NUtil::TRcu<std::vector<sockaddr_in>> addrsRef_;
    };
}
