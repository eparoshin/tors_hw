#pragma once

#include <concepts>
#include <mutex>
#include <shared_mutex>
#include <memory>

namespace NUtil {
    template <typename T>
    class TRcu {
    public:
        using TRef = std::shared_ptr<const T>;

        template <std::convertible_to<T> TRef>
        void Set(TRef&& ref) {
            std::unique_lock ul(m_);
            ref_ = std::make_shared<T>(std::forward<TRef>(ref));
        }

        TRef Aquire() const {
            std::shared_lock sl(m_);
            return ref_;
        }
    private:
        mutable std::shared_mutex m_;
        TRef ref_;
    };
}
