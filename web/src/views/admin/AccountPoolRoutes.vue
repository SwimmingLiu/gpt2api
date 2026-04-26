<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as poolApi from '@/api/account-pools'
import * as statsApi from '@/api/stats'

const loading = ref(false)
const saving = ref(false)
const routes = ref<poolApi.AccountPoolRoute[]>([])
const pools = ref<poolApi.AccountPool[]>([])
const models = ref<statsApi.Model[]>([])

const dialogVisible = ref(false)
const form = reactive({
  model_id: 0,
  pool_id: 0,
  fallback_pool_id: 0,
  enabled: true,
})

const imageModels = computed(() => models.value.filter((item) => item.type === 'image'))

function poolName(id: number) {
  return pools.value.find((item) => item.id === id)?.name || `#${id}`
}

function modelLabel(id: number) {
  return imageModels.value.find((item) => item.id === id)?.slug || `#${id}`
}

async function loadAll() {
  loading.value = true
  try {
    const [routeData, poolData, modelData] = await Promise.all([
      poolApi.listRoutes(),
      poolApi.listPools(),
      statsApi.listModels(),
    ])
    routes.value = routeData.items || []
    pools.value = poolData.items || []
    models.value = modelData.items || []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, {
    model_id: imageModels.value.find((item) => item.slug === 'gpt-image-2')?.id || 0,
    pool_id: 0,
    fallback_pool_id: 0,
    enabled: true,
  })
  dialogVisible.value = true
}

function openEdit(row: poolApi.AccountPoolRoute) {
  Object.assign(form, {
    model_id: row.model_id,
    pool_id: row.pool_id,
    fallback_pool_id: row.fallback_pool_id,
    enabled: row.enabled,
  })
  dialogVisible.value = true
}

async function saveRoute() {
  saving.value = true
  try {
    await poolApi.putRoute(form.model_id, {
      pool_id: form.pool_id,
      fallback_pool_id: form.fallback_pool_id,
      enabled: form.enabled,
    })
    ElMessage.success('池路由已保存')
    dialogVisible.value = false
    await loadAll()
  } finally {
    saving.value = false
  }
}

async function removeRoute(row: poolApi.AccountPoolRoute) {
  await ElMessageBox.confirm(`确认删除模型 ${modelLabel(row.model_id)} 的池路由?`, '确认删除', { type: 'warning' })
  await poolApi.deleteRoute(row.model_id)
  ElMessage.success('池路由已删除')
  await loadAll()
}

onMounted(() => {
  loadAll()
})
</script>

<template>
  <div class="page-container" v-loading="loading">
    <div class="page-title">池路由管理</div>
    <div class="page-subtitle">
      当前仅建议为 <code>gpt-image-2</code> 配置路由。请求会先命中主池,主池无可用成员时再进入 fallback。
    </div>

    <div class="card-block">
      <div class="flex-between block-head">
        <div>
          <div class="section-title">模型到账号池的映射</div>
          <div class="section-tip">如果未显式建模,运行时会回退到配置中的默认图片池。</div>
        </div>
        <div class="flex-wrap-gap">
          <el-button @click="loadAll">刷新</el-button>
          <el-button type="primary" @click="openCreate">新增路由</el-button>
        </div>
      </div>

      <el-table :data="routes" row-key="model_id">
        <el-table-column prop="model_id" label="Model ID" width="90" />
        <el-table-column label="模型" min-width="160">
          <template #default="{ row }">
            {{ modelLabel(row.model_id) }}
          </template>
        </el-table-column>
        <el-table-column label="主池" min-width="140">
          <template #default="{ row }">
            {{ poolName(row.pool_id) }}
          </template>
        </el-table-column>
        <el-table-column label="Fallback" min-width="140">
          <template #default="{ row }">
            <span v-if="row.fallback_pool_id">{{ poolName(row.fallback_pool_id) }}</span>
            <span v-else class="muted">未配置</span>
          </template>
        </el-table-column>
        <el-table-column label="启用" width="90">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="removeRoute(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dialogVisible" title="池路由" width="560px">
      <el-form label-width="120px">
        <el-form-item label="模型">
          <el-select v-model="form.model_id" style="width: 100%">
            <el-option
              v-for="item in imageModels"
              :key="item.id"
              :label="`${item.slug} (#${item.id})`"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="主池">
          <el-select v-model="form.pool_id" style="width: 100%">
            <el-option
              v-for="item in pools"
              :key="item.id"
              :label="`${item.name} (#${item.id})`"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Fallback">
          <el-select v-model="form.fallback_pool_id" clearable style="width: 100%">
            <el-option
              v-for="item in pools"
              :key="item.id"
              :label="`${item.name} (#${item.id})`"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveRoute">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-subtitle {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  margin-bottom: 16px;
}
.block-head {
  margin-bottom: 12px;
}
.section-title {
  font-size: 16px;
  font-weight: 600;
}
.section-tip {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-top: 4px;
}
.muted {
  color: var(--el-text-color-secondary);
}
</style>
