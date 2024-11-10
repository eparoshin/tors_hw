#pragma once

#include <functional>
#include <utility>

namespace NUtil {
    template <typename TFunc>
    class TDefer {
    public:
        explicit TDefer(TFunc func)
        : func_(std::move(func)) {
        }

        ~TDefer() {
            std::invoke(std::move(func_));
        }
    private:
        TFunc func_;
    };
}
