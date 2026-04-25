<script setup lang="ts">
import type { ManualAccountForm, SelectOption } from './types'

const model = defineModel<ManualAccountForm>({
  required: true,
})

withDefaults(
  defineProps<{
    disabled?: boolean
    proxyOptions?: SelectOption[]
  }>(),
  {
    disabled: false,
    proxyOptions: () => [],
  },
)
</script>

<template>
  <div class="pane">
    <el-form label-width="120px">
      <el-form-item label="邮箱">
        <el-input v-model="model.email" :disabled="disabled" placeholder="user@example.com" />
      </el-form-item>

      <el-form-item label="Access Token">
        <el-input
          v-model="model.auth_token"
          :autosize="{ minRows: 3, maxRows: 5 }"
          :disabled="disabled"
          type="textarea"
        />
      </el-form-item>

      <el-form-item label="Refresh Token">
        <el-input
          v-model="model.refresh_token"
          :autosize="{ minRows: 2, maxRows: 4 }"
          :disabled="disabled"
          type="textarea"
        />
      </el-form-item>

      <el-form-item label="Session Token">
        <el-input
          v-model="model.session_token"
          :autosize="{ minRows: 2, maxRows: 4 }"
          :disabled="disabled"
          type="textarea"
        />
      </el-form-item>

      <el-form-item label="Client ID">
        <el-input v-model="model.client_id" :disabled="disabled" />
      </el-form-item>

      <el-form-item label="账号类型">
        <el-select v-model="model.account_type" :disabled="disabled" style="width: 100%">
          <el-option label="Codex" value="codex" />
          <el-option label="ChatGPT" value="chatgpt" />
          <el-option label="OpenAI" value="openai" />
        </el-select>
      </el-form-item>

      <el-form-item label="套餐">
        <el-select v-model="model.plan_type" :disabled="disabled" style="width: 100%">
          <el-option label="Plus" value="plus" />
          <el-option label="Team" value="team" />
          <el-option label="Free" value="free" />
        </el-select>
      </el-form-item>

      <el-form-item label="默认代理">
        <el-select
          v-model="model.proxy_id"
          clearable
          filterable
          placeholder="不指定"
          :disabled="disabled"
          style="width: 100%"
        >
          <el-option
            v-for="item in proxyOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
            :disabled="item.disabled"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="日额度">
        <el-input-number v-model="model.daily_image_quota" :disabled="disabled" :min="0" :step="10" />
      </el-form-item>

      <el-form-item label="备注">
        <el-input
          v-model="model.notes"
          :autosize="{ minRows: 2, maxRows: 4 }"
          :disabled="disabled"
          type="textarea"
        />
      </el-form-item>
    </el-form>
  </div>
</template>
